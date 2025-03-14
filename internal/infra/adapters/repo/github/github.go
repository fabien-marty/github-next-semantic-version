package repogithub

import (
	"context"
	"log/slog"
	"time"

	"github.com/fabien-marty/github-next-semantic-version/internal/app/repo"
	gh "github.com/google/go-github/v62/github"
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

func (r *Adapter) getPullRequestsSince(state state, base string) ([]*repo.PullRequest, error) {
	logger := slog.Default().With("base", base, "state", string(state))
	listOptionsState := "open"
	if state == merged {
		listOptionsState = "closed"
	}
	listOptions := &gh.PullRequestListOptions{
		State: listOptionsState,
		Base:  base,
		ListOptions: gh.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}
	res := []*repo.PullRequest{}
	for {
		logger.Debug("fetching pull-requests...", slog.Int("page", listOptions.Page))
		prs, resp, err := r.client.PullRequests.List(context.Background(), r.owner, r.repo, listOptions)
		if err != nil {
			return nil, err
		}
		for _, pr := range prs {
			if pr.Number == nil || pr.Title == nil || pr.UpdatedAt == nil || pr.CreatedAt == nil || pr.HTMLURL == nil || pr.Head == nil || pr.Head.Ref == nil || pr.User == nil || pr.User.Login == nil || pr.User.HTMLURL == nil {
				continue
			}
			if state == "merged" {
				if pr.MergedAt == nil {
					continue
				}
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
			res = append(res, &repo.PullRequest{
				Number:      *pr.Number,
				Title:       *pr.Title,
				MergedAt:    mergedAt,
				Labels:      labels,
				Branch:      *pr.Head.Ref,
				Url:         *pr.HTMLURL,
				AuthorLogin: *pr.User.Login,
				AuthorUrl:   *pr.User.HTMLURL,
			})
		}
		if resp.NextPage == 0 {
			break
		}
		listOptions.Page = resp.NextPage
	}
	logger.Debug("pull-requests fetched", slog.Int("count", len(res)))
	return res, nil
}

func (r *Adapter) GetPullRequestsSince(base string, onlyMerged bool) (res []*repo.PullRequest, err error) {
	if onlyMerged {
		return r.getPullRequestsSince(merged, base)
	}
	opened, err := r.getPullRequestsSince(open, base)
	if err != nil {
		return nil, err
	}
	merged, err := r.getPullRequestsSince(merged, base)
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
