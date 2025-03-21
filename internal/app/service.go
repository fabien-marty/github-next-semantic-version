package app

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"regexp"
	"slices"
	"sort"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/changelog"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/git"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/repo"
)

// errNoTags is the error returned when no tag is found
var errNoTags = errors.New("no existing tag found")
var ErrNoRelease = errors.New("no need to create a release")

const (
	nothing             = "nothing"
	major               = "major"
	minor               = "minor"
	patch               = "patch"
	defaultFirstVersion = "v0.0.0"
)

// Service is the main application service
type Service struct {
	Config      Config
	RepoAdapter repo.Port
	GitAdapter  git.Port
	logger      *slog.Logger
}

// NewService creates a new Service
func NewService(config Config, repoAdapter repo.Port, gitAdapter git.Port) *Service {
	return &Service{
		Config:      config,
		RepoAdapter: repoAdapter,
		GitAdapter:  gitAdapter,
		logger:      slog.Default(),
	}
}

// getContainedTags returns the list of tags contained by the branch set after the given time
// (the list from the adapter is optionally filtered by the tag-regex configuration,
// the branch can be empty, since can be empty)
// the returned slice is sorted by (ascending) semantic version
// tags without bad semantic version are ignored
func (s *Service) getContainedTagsSingleBranch(branch string, since *time.Time) ([]*git.Tag, error) {
	res, err := s.GitAdapter.GetContainedTags(branch)
	if err != nil {
		return nil, err
	}
	regex, err := regexp.Compile(s.Config.TagRegex)
	if err != nil {
		return res, fmt.Errorf("can't compile the regex %s: %w", s.Config.TagRegex, err)
	}
	res = slices.DeleteFunc(res, func(tag *git.Tag) bool {
		if !regex.MatchString(tag.Name) {
			s.logger.Debug("tag doesn't match the regex => ignoring", slog.String("name", tag.Name), slog.String("regex", s.Config.TagRegex))
			return true
		}
		if tag.Semver == nil {
			s.logger.Debug("tag doesn't have a semantic version => ignoring", slog.String("name", tag.Name))
			return true
		}
		if tag.Semver.Prerelease() != "" {
			s.logger.Debug("tag is a prelease => ignoring", slog.String("name", tag.Name))
			return true
		}
		if since != nil && (tag.Time.Before(*since) || tag.Time.Equal(*since)) {
			s.logger.Debug("tag too old => ignoring", slog.String("name", tag.Name), slog.String("time", tag.Time.Format(time.RFC3339)))
			return true
		}
		return false
	})
	sort.Slice(res, func(i, j int) bool {
		return res[i].LessThan(res[j])
	})
	return res, nil
}

func (s *Service) getContainedTags(branches []string, since *time.Time) ([]*git.Tag, error) {
	res := []*git.Tag{}
	for _, branch := range branches {
		tags, err := s.getContainedTagsSingleBranch(branch, since)
		if err != nil {
			return nil, err
		}
		res = append(res, tags...)
	}
	return res, nil
}

// getPullRequests returns the list of PRs merged since the given time
// (the list from the adapter is optionally filtered by the PullRequestIgnoreLabels configuration)
// the returned slice is sorted by (ascending) mergedAt
func (s *Service) getPullRequestsSingleBranch(branch string, since *time.Time, onlyMerged bool) ([]*repo.PullRequest, error) {
	prs, err := s.RepoAdapter.GetPullRequests(branch, onlyMerged)
	prs = slices.DeleteFunc(prs, func(pr *repo.PullRequest) bool {
		mergedAt := pr.MergedAt
		if since != nil && mergedAt != nil && mergedAt.Before(*since) {
			return true
		}
		if pr.IsIgnored(s.Config.PullRequestIgnoreLabels) {
			s.logger.Debug("the pr has an ignored label", slog.Int("number", pr.Number))
			return true
		}
		if len(s.Config.PullRequestMustHaveLabels) > 0 {
			if !pr.HasOneOfTheseLabels(s.Config.PullRequestMustHaveLabels) {
				s.logger.Debug("the pr doesn't have one of the required labels", slog.Int("number", pr.Number))
				return true
			}
		}
		return false
	})
	sort.Slice(prs, func(i, j int) bool {
		if prs[i].MergedAt == nil {
			return false
		}
		if prs[j].MergedAt == nil {
			return true
		}
		return prs[i].MergedAt.Before(*prs[j].MergedAt)
	})
	return prs, err
}

func (s *Service) getPullRequests(branches []string, since *time.Time, onlyMerged bool) ([]*repo.PullRequest, error) {
	res := []*repo.PullRequest{}
	for _, branch := range branches {
		prs, err := s.getPullRequestsSingleBranch(branch, since, onlyMerged)
		if err != nil {
			return nil, err
		}
		res = append(res, prs...)
	}
	return res, nil
}

// getLatestSemanticNonPrereleaseTag returns the latest semantic (non-prerelease) tag contained by the branch
// If no tag is found, it returns ErrNoTags
func (s *Service) getLatestSemanticNonPrereleaseTag(branches []string) (*git.Tag, error) {
	tags, err := s.getContainedTags(branches, nil)
	if err != nil {
		return nil, fmt.Errorf("can't get the list of tags contained by %s: %w", branches, err)
	}
	slog.Debug(fmt.Sprintf("%d tags found", len(tags)))
	if len(tags) == 0 {
		return nil, errNoTags
	}
	return tags[len(tags)-1], nil
}

// GetNextVersion returns the next semantic version based on the branch and the PRs merged since the last tag + PRs still opened (if onlyMerged is false)
func (s *Service) GetNextVersion(branches []string, onlyMerged bool, dontIncrementIfNoPR bool) (oldVersion string, newVersion string, consideredPullRequests []*repo.PullRequest, err error) {
	logger := s.logger
	latestTag, err := s.getLatestSemanticNonPrereleaseTag(branches)
	if err == errNoTags {
		logger.Warn("no tag found => let's use the default first version")
		latestTag = git.NewTag(defaultFirstVersion, time.Unix(0, 0))
	} else if err != nil {
		return "", "", nil, err
	}
	logger.Debug(fmt.Sprintf("latest semantic (non-prerelease) tag found: %s (date: %s)", latestTag.Name, latestTag.Time.Format(time.RFC3339)))
	prs, err := s.getPullRequests(branches, &latestTag.Time, onlyMerged)
	if err != nil {
		return "", "", nil, err
	}
	logger.Debug(fmt.Sprintf("%d PRs to consider", len(prs)))
	increment := nothing
	for _, pr := range prs {
		logger := logger.With(slog.Int("number", pr.Number), slog.String("title", pr.Title), slog.Bool("merged", pr.MergedAt != nil))
		if pr.MergedAt != nil {
			logger = logger.With(slog.String("mergedAt", pr.MergedAt.Format(time.RFC3339)))
		}
		consideredPullRequests = append(consideredPullRequests, pr)
		if pr.IsMajor(s.Config.PullRequestMajorLabels) {
			logger.Debug("major PR found => break")
			increment = major
			break
		} else if pr.IsMinor(s.Config.PullRequestMinorLabels) {
			logger.Debug("minor PR found")
			if increment == nothing || increment == patch {
				increment = minor
			}
		} else {
			logger.Debug("patch PR found")
			if increment == nothing {
				increment = patch
			}
		}
	}
	switch increment {
	case nothing:
		if dontIncrementIfNoPR {
			logger.Debug("we found no PR (or they are all ignored) and DontIncrementIfNoPr is true => let's not increment the version")
			return latestTag.Name, latestTag.Name, consideredPullRequests, nil
		} else {
			logger.Debug("we found no PR (or they are all ignored) and DontIncrementIfNoPr is false => let's increment the patch number")
			return latestTag.Name, latestTag.NewName(latestTag.Semver.IncPatch()), consideredPullRequests, nil
		}
	case major:
		logger.Debug("we found at least one MAJOR PR => let's increment the major number")
		return latestTag.Name, latestTag.NewName(latestTag.Semver.IncMajor()), consideredPullRequests, nil
	case minor:
		logger.Debug("we found at least one MINOR PR => let's increment the minor number")
		return latestTag.Name, latestTag.NewName(latestTag.Semver.IncMinor()), consideredPullRequests, nil
	case patch:
		logger.Debug("we found some PRs but we didn't find MAJOR or MINOR PRs => let's increment the patch number")
		return latestTag.Name, latestTag.NewName(latestTag.Semver.IncPatch()), consideredPullRequests, nil
	default:
		panic(fmt.Sprintf("unknown increment value: %s", increment))
	}
}

func (s *Service) getReleaseBodyFromPRs(prs []*repo.PullRequest, bodyTemplate *template.Template) (string, error) {
	var body bytes.Buffer
	err := bodyTemplate.Execute(&body, prs)
	if err != nil {
		return "", fmt.Errorf("can't execute the template: %w on pr: %+v", err, prs)
	}
	return body.String(), nil
}

func (s *Service) CreateNextRelease(branches []string, dontIncrementIfNoPR bool, draft bool, bodyTemplateString string) (newTag string, err error) {
	if len(branches) != 1 {
		return "", errors.New("only one branch is supported")
	}
	oldTag, newTag, prs, err := s.GetNextVersion(branches, true, dontIncrementIfNoPR)
	if err != nil {
		return "", err
	}
	if oldTag == newTag {
		return "", ErrNoRelease
	}
	bodyTemplate := template.New("body")
	bodyTemplate, err = bodyTemplate.Parse(bodyTemplateString)
	if err != nil {
		return "", fmt.Errorf("can't parse the template: %w", err)
	}
	body, err := s.getReleaseBodyFromPRs(prs, bodyTemplate)
	if err != nil {
		return "", fmt.Errorf("can't create the release body: %w", err)
	}
	return newTag, s.RepoAdapter.CreateRelease(branches[0], newTag, body, draft)
}

func (s *Service) GenerateChangelog(branches []string, onlyMerged bool, future bool, sinceTag string, changelogTemplateString string) (string, error) {
	if len(branches) == 0 {
		return "", errors.New("at least one branch is required")
	}
	var since *time.Time = nil
	if sinceTag == "LATEST" {
		if !future {
			return "", errors.New("sinceTag=LATEST is only compatible with future=true")
		}
		latestTag, err := s.getLatestSemanticNonPrereleaseTag(branches)
		if err != nil {
			if err != errNoTags {
				return "", err
			}
		} else {
			since = &latestTag.Time
		}
	}
	changelogTemplate := template.New("changelog").Funcs(sprig.FuncMap())
	changelogTemplate, err := changelogTemplate.Parse(changelogTemplateString)
	if err != nil {
		return "", fmt.Errorf("can't parse the template: %w", err)
	}
	tags, err := s.getContainedTags(branches, since)
	if err != nil {
		return "", err
	}
	if sinceTag != "" {
		for i, tag := range tags {
			if tag.Name == sinceTag {
				since = &tag.Time
				if i >= len(tags)-1 {
					tags = nil
				} else {
					tags = tags[i+1:]
				}
				break
			}
		}
	}
	prs, err := s.getPullRequests(branches, since, onlyMerged)
	if err != nil {
		return "", err
	}
	changelog := changelog.New(tags, prs, changelog.Config{
		MinimalDelayInSeconds:   s.Config.MinimalDelayInSeconds,
		Future:                  future,
		RepoOwner:               s.Config.RepoOwner,
		RepoName:                s.Config.RepoName,
		PullRequestIgnoreLabels: s.Config.PullRequestIgnoreLabels,
	})
	var body bytes.Buffer
	err = changelogTemplate.Execute(&body, changelog)
	if err != nil {
		return "", fmt.Errorf("can't execute the template: %w on changelog object: %+v", err, changelog)
	}
	return body.String(), nil
}
