package repogithub

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/fabien-marty/github-next-semantic-version/internal/app/repo"
	gh "github.com/google/go-github/v70/github"
)

var _ repo.Port = &Adapter{}

type state string

const (
	open   state = "opened"
	merged state = "merged"
)

type AdapterOptions struct {
	Token string
}

type Adapter struct {
	opts   AdapterOptions
	client *gh.Client
	owner  string
	repo   string
}

func NewAdapter(owner string, repo string, opts AdapterOptions) *Adapter {
	client := gh.NewClient(nil)
	if opts.Token != "" {
		client = client.WithAuthToken(opts.Token)
	}
	return &Adapter{
		client: client,
		opts:   opts,
		owner:  owner,
		repo:   repo,
	}
}

func (r *Adapter) createPullRequestFromGhPr(pr *gh.PullRequest) *repo.PullRequest {
	if pr.Number == nil || pr.Title == nil || pr.UpdatedAt == nil || pr.CreatedAt == nil || pr.HTMLURL == nil || pr.Head == nil || pr.Head.Ref == nil || pr.User == nil || pr.User.Login == nil || pr.User.HTMLURL == nil {
		return nil
	}
	labels := []string{}
	for _, label := range pr.Labels {
		if label.Name == nil {
			continue
		}
		labels = append(labels, *label.Name)
	}
	var mergedAt *time.Time
	if pr.MergedAt != nil {
		mergedAt = pr.MergedAt.GetTime()
	}
	var updatedAt *time.Time
	if pr.UpdatedAt != nil {
		updatedAt = pr.UpdatedAt.GetTime()
	}
	return &repo.PullRequest{
		Number:      *pr.Number,
		Title:       *pr.Title,
		MergedAt:    mergedAt,
		UpdatedAt:   updatedAt,
		Labels:      labels,
		Branch:      *pr.Head.Ref,
		Url:         *pr.HTMLURL,
		AuthorLogin: *pr.User.Login,
		AuthorUrl:   *pr.User.HTMLURL,
	}
}

func (r *Adapter) getPullRequestsSince(state state, base string, sort string, usePagination bool) ([]*repo.PullRequest, error) {
	listOptionsState := "open"
	if state == merged {
		listOptionsState = "closed"
	}
	listOptions := &gh.PullRequestListOptions{
		State: listOptionsState,
		Base:  base,
		Sort:  sort,
		ListOptions: gh.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}
	logger := slog.Default().With("base", base, "state", string(state), "sort", sort, "page", 1)
	res := []*repo.PullRequest{}
	for {
		logger := logger.With("page", listOptions.Page)
		logger.Debug("fetching pull-requests...")
		prs, resp, err := r.client.PullRequests.List(context.Background(), r.owner, r.repo, listOptions)
		if err != nil {
			return nil, err
		}
		for _, pr := range prs {
			pro := r.createPullRequestFromGhPr(pr)
			if pro == nil {
				continue
			}
			if state == "merged" && pro.MergedAt == nil {
				continue
			}
			res = append(res, pro)
		}
		if !usePagination || resp.NextPage == 0 {
			break
		}
		listOptions.Page = resp.NextPage
	}
	logger.Debug("pull-requests fetched", slog.Int("count", len(res)))
	return res, nil
}

func sortPRByUpdatedAt(a, b *repo.PullRequest) int {
	if (a == nil) || (b == nil) {
		panic("can't be nil")
	}
	if (a.UpdatedAt == nil) || (b.UpdatedAt == nil) {
		panic("UpdateAt can't be nil")
	}
	if a.UpdatedAt.Before(*b.UpdatedAt) {
		return -1
	} else {
		return 1
	}
}

func (r *Adapter) GetLastUpdatedPullRequests(base string, onlyMerged bool) ([]*repo.PullRequest, error) {
	if onlyMerged {
		return r.getPullRequestsSince(merged, base, "updated", false)
	}
	opened, err := r.getPullRequestsSince(open, base, "updated", false)
	if err != nil {
		return nil, err
	}
	merged, err := r.getPullRequestsSince(merged, base, "updated", false)
	if err != nil {
		return nil, err
	}
	tmp := []*repo.PullRequest{}
	tmp = append(tmp, opened...)
	tmp = append(tmp, merged...)
	slices.SortFunc(tmp, sortPRByUpdatedAt)
	slices.Reverse(tmp)
	return tmp, nil
}

func (r *Adapter) GetPullRequests(base string, onlyMerged bool) (res []*repo.PullRequest, err error) {
	if onlyMerged {
		return r.getPullRequestsSince(merged, base, "created", true)
	}
	opened, err := r.getPullRequestsSince(open, base, "created", true)
	if err != nil {
		return nil, err
	}
	merged, err := r.getPullRequestsSince(merged, base, "created", true)
	if err != nil {
		return nil, err
	}
	return append(opened, merged...), nil
}

func (r *Adapter) CreateRelease(base string, tagName string, body string, draft bool) error {
	makeLatestAsString := "true"
	prerelease := false
	_, _, err := r.client.Repositories.CreateRelease(context.Background(), r.owner, r.repo, &gh.RepositoryRelease{
		TagName:         &tagName,
		TargetCommitish: &base,
		Name:            &tagName,
		Body:            &body,
		Draft:           &draft,
		Prerelease:      &prerelease,
		MakeLatest:      &makeLatestAsString,
	})
	if err != nil {
		return err
	}
	return nil
}
