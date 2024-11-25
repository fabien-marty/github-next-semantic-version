package app

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"strings"
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
	config      Config
	repoService *repo.Service
	gitService  *git.Service
	logger      *slog.Logger
}

// New creates a new Service
func New(config Config, repoService *repo.Service, gitService *git.Service) *Service {
	return &Service{
		config:      config,
		repoService: repoService,
		gitService:  gitService,
		logger:      slog.Default(),
	}
}

func (s *Service) GuessDefaultBranch() string {
	return s.gitService.GuessDefaultBranch()
}

// getLatestTag returns the latest semantic (non-prerelease) tag contained by the branch
// If no tag is found, it returns ErrNoTags
func (s *Service) getLatestTag(branches []string, ignorePrereleases bool, ignoreNonSemantic bool) (*git.Tag, error) {
	tags, err := s.gitService.GetTags(branches, nil, s.config.TagRegex, ignorePrereleases, ignoreNonSemantic)
	if err != nil {
		return nil, fmt.Errorf("can't get the list of tags contained by %s: %w", branches, err)
	}
	slog.Debug(fmt.Sprintf("%d tags found", len(tags)))
	if len(tags) == 0 {
		return nil, errNoTags
	}
	return tags[len(tags)-1], nil
}

func (s *Service) getNextVersion(branches []string, onlyMerged bool, dontIncrementIfNoPR bool, ignorePrereleases bool) (oldTag *git.Tag, newTag *git.Tag, consideredPullRequests []*repo.PullRequest, err error) {
	logger := s.logger
	latestTag, err := s.getLatestTag(branches, ignorePrereleases, true)
	if err == errNoTags {
		logger.Warn("no tag found => let's use the default first version")
		dummyTime := time.Unix(0, 0)
		latestTag = git.NewTag(defaultFirstVersion, &dummyTime)
	} else if err != nil {
		return nil, nil, nil, err
	} else {
		logger.Debug(fmt.Sprintf("latest semantic (non-prerelease) tag found: %s (date: %s)", latestTag.Name, latestTag.Time.Format(time.RFC3339)))
	}
	prs, err := s.repoService.GetPullRequests(branches, latestTag.Time, onlyMerged, s.config.PullRequestIgnoreLabels, s.config.PullRequestMustHaveLabels)
	if err != nil {
		return nil, nil, nil, err
	}
	logger.Debug(fmt.Sprintf("%d PRs to consider", len(prs)))
	increment := nothing
	for _, pr := range prs {
		logger := logger.With(slog.Int("number", pr.Number), slog.String("title", pr.Title), slog.Bool("merged", pr.MergedAt != nil))
		if pr.MergedAt != nil {
			logger = logger.With(slog.String("mergedAt", pr.MergedAt.Format(time.RFC3339)))
		}
		consideredPullRequests = append(consideredPullRequests, pr)
		if pr.IsMajor(s.config.PullRequestMajorLabels) {
			logger.Debug("major PR found => break")
			increment = major
			break
		} else if pr.IsMinor(s.config.PullRequestMinorLabels) {
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
			return latestTag, latestTag, consideredPullRequests, nil
		} else {
			logger.Debug("we found no PR (or they are all ignored) and DontIncrementIfNoPr is false => let's increment the patch number")
			return latestTag, git.NewTagIncrementingPatch(latestTag), consideredPullRequests, nil
		}
	case major:
		logger.Debug("we found at least one MAJOR PR => let's increment the major number")
		return latestTag, git.NewTagIncrementingMajor(latestTag), consideredPullRequests, nil
	case minor:
		logger.Debug("we found at least one MINOR PR => let's increment the minor number")
		return latestTag, git.NewTagIncrementingMinor(latestTag), consideredPullRequests, nil
	case patch:
		logger.Debug("we found some PRs but we didn't find MAJOR or MINOR PRs => let's increment the patch number")
		return latestTag, git.NewTagIncrementingPatch(latestTag), consideredPullRequests, nil
	default:
		panic(fmt.Sprintf("unknown increment value: %s", increment))
	}
}

func (s *Service) GetNextVersion(branches []string, onlyMerged bool, dontIncrementIfNoPR bool, ignorePrereleases bool) (oldTagName string, newTagName string, err error) {
	oldTag, newTag, _, err := s.getNextVersion(branches, onlyMerged, dontIncrementIfNoPR, ignorePrereleases)
	if err != nil {
		return "", "", err
	}
	return oldTag.Name, newTag.Name, err
}

func (s *Service) CreateNextRelease(branches []string, dontIncrementIfNoPR bool, draft bool, changelogTemplateString string, ignorePrereleases bool) (newTagName string, err error) {
	if len(branches) != 1 {
		return "", errors.New("only one branch is supported")
	}
	oldTag, newTag, prs, err := s.getNextVersion(branches, true, dontIncrementIfNoPR, ignorePrereleases)
	if err != nil {
		return "", err
	}
	if oldTag.Name == newTag.Name {
		return "", ErrNoRelease
	}
	changelogTemplate := template.New("changelog").Funcs(sprig.FuncMap())
	changelogTemplate, err = changelogTemplate.Parse(changelogTemplateString)
	if err != nil {
		return "", fmt.Errorf("can't parse the template: %w", err)
	}
	tags := []*git.Tag{oldTag}
	changelog := changelog.New(tags, prs, changelog.Config{
		MinimalDelayInSeconds: s.config.MinimalDelayInSeconds,
		Future:                true,
		RepoOwner:             s.config.RepoOwner,
		RepoName:              s.config.RepoName,
		HideMainHeader:        true,
		HideSectionHeaders:    true,
	})
	var body bytes.Buffer
	err = changelogTemplate.Execute(&body, changelog)
	if err != nil {
		return "", fmt.Errorf("can't execute the template: %w on changelog object: %+v", err, changelog)
	}
	bodyAsString := strings.TrimSpace(body.String()) + "\n"
	return newTag.Name, s.repoService.CreateRelease(branches[0], newTag.Name, bodyAsString, draft)
}

func (s *Service) GenerateChangelog(branches []string, onlyMerged bool, future bool, sinceTag string, changelogTemplateString string, ignorePrereleases bool) (string, error) {
	if len(branches) == 0 {
		return "", errors.New("at least one branch is required")
	}
	var since *time.Time = nil
	if sinceTag == "LATEST" {
		if !future {
			return "", errors.New("sinceTag=LATEST is only compatible with future=true")
		}
		latestTag, err := s.getLatestTag(branches, ignorePrereleases, true)
		if err != nil {
			if err != errNoTags {
				return "", err
			}
		} else {
			since = latestTag.Time
		}
	}
	changelogTemplate := template.New("changelog").Funcs(sprig.FuncMap())
	changelogTemplate, err := changelogTemplate.Parse(changelogTemplateString)
	if err != nil {
		return "", fmt.Errorf("can't parse the template: %w", err)
	}
	tags, err := s.gitService.GetTags(branches, since, s.config.TagRegex, ignorePrereleases, true)
	if err != nil {
		return "", err
	}
	if sinceTag != "" {
		for i, tag := range tags {
			if tag.Name == sinceTag {
				since = tag.Time
				if i >= len(tags)-1 {
					tags = nil
				} else {
					tags = tags[i+1:]
				}
				break
			}
		}
	}
	prs, err := s.repoService.GetPullRequests(branches, since, onlyMerged, s.config.PullRequestIgnoreLabels, s.config.PullRequestMustHaveLabels)
	if err != nil {
		return "", err
	}
	changelog := changelog.New(tags, prs, changelog.Config{
		MinimalDelayInSeconds: s.config.MinimalDelayInSeconds,
		Future:                future,
		RepoOwner:             s.config.RepoOwner,
		RepoName:              s.config.RepoName,
	})
	var body bytes.Buffer
	err = changelogTemplate.Execute(&body, changelog)
	if err != nil {
		return "", fmt.Errorf("can't execute the template: %w on changelog object: %+v", err, changelog)
	}
	bodyAsString := strings.TrimSpace(body.String()) + "\n"
	return bodyAsString, nil
}
