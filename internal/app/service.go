package app

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"regexp"
	"time"

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
	config      Config
	repoAdapter repo.Port
	gitAdapter  git.Port
	logger      *slog.Logger
}

// NewService creates a new Service
func NewService(config Config, repoAdapter repo.Port, gitAdapter git.Port) *Service {
	return &Service{
		config:      config,
		repoAdapter: repoAdapter,
		gitAdapter:  gitAdapter,
		logger:      slog.Default(),
	}
}

// getContainedTags returns the list of tags contained by the branch
// (the list from the adapter is optionally filtered by the tag-regex configuration)
func (s *Service) getContainedTags(branch string) ([]*git.Tag, error) {
	res, err := s.gitAdapter.GetContainedTags(branch)
	if err != nil {
		return nil, err
	}
	if s.config.TagRegex == "" {
		return res, nil
	}
	var filtered []*git.Tag
	regex, err := regexp.Compile(s.config.TagRegex)
	if err != nil {
		return res, fmt.Errorf("can't compile the regex %s: %w", s.config.TagRegex, err)
	}
	for _, tag := range res {
		if regex.MatchString(tag.Name) {
			filtered = append(filtered, tag)
		}
	}
	return filtered, nil
}

// getLatestSemanticNonPrereleaseTag returns the latest semantic (non-prerelease) tag contained by the branch
// If no tag is found, it returns ErrNoTags
func (s *Service) getLatestSemanticNonPrereleaseTag(branch string) (*git.Tag, error) {
	logger := s.logger.With(slog.String("branch", branch))
	tags, err := s.getContainedTags(branch)
	if err != nil {
		return nil, fmt.Errorf("can't get the list of tags contained by %s: %w", branch, err)
	}
	slog.Debug(fmt.Sprintf("%d tags found", len(tags)))
	if len(tags) == 0 {
		return nil, errNoTags
	}
	var res *git.Tag = nil
	for _, tag := range tags {
		lgr := logger.With(slog.String("name", tag.Name), slog.String("date", tag.Time.Format(time.RFC3339)))
		if tag.Semver == nil {
			lgr.Debug("can't compute a semver version from the tag name => ignoring")
			continue
		}
		if tag.Semver.Prerelease() != "" {
			lgr.Debug("found a pre-release semver version => ignoring")
			continue
		}
		if res == nil || res.LessThan(tag) {
			lgr.Debug("temporary selecting the tag")
			res = tag
		} else {
			lgr.Debug("ignoring this tag as we have a more recent")
		}
	}
	if res == nil {
		return nil, fmt.Errorf("no semantic version tags contained by branch %s", branch)
	}
	return res, nil
}

// GetNextVersion returns the next semantic version based on the branch and the PRs merged since the last tag + PRs still opened (if onlyMerged is false)
func (s *Service) GetNextVersion(branch string, onlyMerged bool, dontIncrementIfNoPR bool) (oldVersion string, newVersion string, consideredPullRequests []repo.PullRequest, err error) {
	logger := s.logger
	latestTag, err := s.getLatestSemanticNonPrereleaseTag(branch)
	if err == errNoTags {
		logger.Warn("no tag found => let's use the default first version")
		latestTag = git.NewTag(defaultFirstVersion, time.Unix(0, 0))
	} else if err != nil {
		return "", "", nil, err
	}
	logger.Debug(fmt.Sprintf("latest semantic (non-prerelease) tag found: %s (date: %s)", latestTag.Name, latestTag.Time.Format(time.RFC3339)))
	prs, err := s.repoAdapter.GetPullRequestsSince(branch, latestTag.Time, onlyMerged)
	if err != nil {
		return "", "", nil, err
	}
	logger.Debug(fmt.Sprintf("%d PRs to consider", len(prs)))
	increment := nothing
	pullRequestConfig := s.config.PullRequestConfig()
	for _, pr := range prs {
		logger := logger.With(slog.Int("number", pr.Number), slog.String("title", pr.Title), slog.Bool("merged", pr.MergedAt != nil))
		if pr.MergedAt != nil {
			logger = logger.With(slog.String("mergedAt", pr.MergedAt.Format(time.RFC3339)))
			if latestTag.Time.Add(time.Second * time.Duration(s.config.MinimalDelayInSeconds)).After(*pr.MergedAt) {
				logger.Debug("PR merged too soon, lightweight tag probably used => ignoring this PR")
				continue
			}
		}
		if pr.IsIgnored(pullRequestConfig) {
			logger.Debug("ignored PR found")
			continue
		}
		consideredPullRequests = append(consideredPullRequests, pr)
		if pr.IsMajor(pullRequestConfig) {
			logger.Debug("major PR found => break")
			increment = major
			break
		} else if pr.IsMinor(pullRequestConfig) {
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

func (s *Service) getReleaseBodyFromPRs(prs []repo.PullRequest, bodyTemplate *template.Template) (string, error) {
	var body bytes.Buffer
	err := bodyTemplate.Execute(&body, prs)
	if err != nil {
		return "", fmt.Errorf("can't execute the template: %w on pr: %+v", err, prs)
	}
	return body.String(), nil
}

func (s *Service) CreateNextRelease(branch string, dontIncrementIfNoPR bool, draft bool, bodyTemplateString string) (newTag string, err error) {
	oldTag, newTag, prs, err := s.GetNextVersion(branch, true, dontIncrementIfNoPR)
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
	return newTag, s.repoAdapter.CreateRelease(branch, newTag, body, draft)
}
