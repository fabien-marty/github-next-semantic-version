package app

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/fabien-marty/github-next-semantic-version/internal/app/git"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/repo"
)

// ErrNoTags is the error returned when no tag is found
var ErrNoTags = errors.New("no existing tag found")

const (
	major               = "major"
	minor               = "minor"
	patch               = "patch"
	defaultFirstVersion = "v0.0.1"
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

// getLatestSemanticNonPrereleaseTag returns the latest semantic (non-prerelease) tag contained by the branch
// If no tag is found, it returns ErrNoTags
func (s *Service) getLatestSemanticNonPrereleaseTag(branch string) (*git.Tag, error) {
	logger := s.logger.With(slog.String("branch", branch))
	tags, err := s.gitAdapter.GetContainedTags(branch)
	if err != nil {
		return nil, fmt.Errorf("can't get the list of tags contained by %s: %w", branch, err)
	}
	slog.Debug(fmt.Sprintf("%d tags found", len(tags)))
	if len(tags) == 0 {
		return nil, ErrNoTags
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
func (s *Service) GetNextVersion(branch string, onlyMerged bool) (string, error) {
	logger := s.logger
	latestTag, err := s.getLatestSemanticNonPrereleaseTag(branch)
	if err == ErrNoTags {
		logger.Warn("no tag found => let's return the default first version")
		return defaultFirstVersion, nil
	} else if err != nil {
		return "", err
	}
	logger.Info(fmt.Sprintf("latest semantic (non-prerelease) tag found: %s (date: %s)", latestTag.Name, latestTag.Time.Format(time.RFC3339)))
	prs, err := s.repoAdapter.GetPullRequestsSince(branch, latestTag.Time, onlyMerged)
	if err != nil {
		return "", err
	}
	logger.Debug(fmt.Sprintf("%d PRs to consider", len(prs)))
	increment := patch
	pullRequestConfig := s.config.PullRequestConfig()
	for _, pr := range prs {
		logger := logger.With(slog.Int("number", pr.Number), slog.String("title", pr.Title), slog.Bool("merged", pr.MergedAt != nil))
		if pr.MergedAt != nil {
			logger = logger.With(slog.String("mergedAt", pr.MergedAt.Format(time.RFC3339)))
		}
		if pr.IsMajor(pullRequestConfig) {
			logger.Debug("major PR found => break")
			increment = major
			break
		} else if pr.IsMinor(pullRequestConfig) {
			logger.Debug("minor PR found")
			increment = minor
			continue
		}
		logger.Debug("patch PR found")
	}
	switch increment {
	case major:
		logger.Info("we found at least one MAJOR PR => let's increment the major number")
		return latestTag.NewName(latestTag.Semver.IncMajor()), nil
	case minor:
		logger.Info("we found at least one MINOR PR => let's increment the minor number")
		return latestTag.NewName(latestTag.Semver.IncMinor()), nil
	case patch:
		logger.Info("we didn't find MAJOR or MINOR PRs => let's increment the patch number")
		return latestTag.NewName(latestTag.Semver.IncPatch()), nil
	default:
		return "", fmt.Errorf("unknown increment %s", increment)
	}
}
