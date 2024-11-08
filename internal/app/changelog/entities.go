package changelog

import (
	"log/slog"
	"slices"
	"sort"
	"time"

	"github.com/fabien-marty/github-next-semantic-version/internal/app/git"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/repo"
)

type Config struct {
	MinimalDelayInSeconds   int
	Future                  bool
	RepoOwner               string
	RepoName                string
	PullRequestIgnoreLabels []string
}

type Section struct {
	Tag *git.Tag            // Tag (or nil if this is the "future" section)
	Prs []*repo.PullRequest // List of pull-requests "contained" by this tag
}

type Changelog struct {
	Sections  []*Section
	RepoOwner string // Repository owner name (organization)
	RepoName  string // Repository name (without owner/organization part)
}

func (c *Changelog) ReversedSections() []*Section {
	reversed := make([]*Section, len(c.Sections))
	copy(reversed, c.Sections)
	slices.Reverse(reversed)
	return reversed
}

func (c *Changelog) GetFuturePrs() []*repo.PullRequest {
	for _, section := range c.Sections {
		if section.Tag == nil {
			return section.Prs
		}
	}
	return nil
}

func (cs *Section) GetPrsWithOneOfTheseLabels(labels []interface{}) []*repo.PullRequest {
	prs := make([]*repo.PullRequest, 0)
	for _, pr := range cs.Prs {
		selected := false
		for _, prLabel := range pr.Labels {
			for _, label := range labels {
				if label == prLabel {
					selected = true
					break
				}
			}
			if selected {
				break
			}
		}
		if selected {
			prs = append(prs, pr)
		}
	}
	return prs
}

func (cs *Section) GetPrsWithNoneOfTheseLabels(labels []interface{}) []*repo.PullRequest {
	prs := make([]*repo.PullRequest, 0)
	for _, pr := range cs.Prs {
		selected := true
		for _, prLabel := range pr.Labels {
			for _, label := range labels {
				if label == prLabel {
					selected = false
					break
				}
			}
			if !selected {
				break
			}
		}
		if selected {
			prs = append(prs, pr)
		}
	}
	return prs
}

// isPullRequestIncludedInThisSegment returns true if the given pr was merged after tag1 and before tag2
// (minimalDelayInSeconds is used to reject some PR when using lightweight tags)
func isPullRequestIncludedInThisSegment(pr *repo.PullRequest, tag1 *git.Tag, tag2 *git.Tag, minimalDelayInSeconds int) bool {
	if pr.MergedAt == nil {
		// not merged PR
		return tag2 == nil // true only if tag2 is the "future" tag
	}
	if tag1 != nil && pr.MergedAt.Before(tag1.Time.Add(time.Second*time.Duration(minimalDelayInSeconds))) {
		return false
	}
	if tag2 == nil { //"future" tag
		return true
	}
	if tag2.Time.Add(time.Second * time.Duration(minimalDelayInSeconds)).Before(*pr.MergedAt) {
		return false
	}
	return true
}

func New(tags []*git.Tag, prs []*repo.PullRequest, config Config) *Changelog {
	logger := slog.Default()
	sections := []*Section{}
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Time.Before(tags[j].Time)
	})
	if config.Future {
		tags = append(tags, nil)
	}
	sort.Slice(prs, func(i, j int) bool {
		if prs[i].MergedAt == nil {
			return false
		}
		if prs[j].MergedAt == nil {
			return true
		}
		return prs[i].MergedAt.Before(*prs[j].MergedAt)
	})
	var previousTag *git.Tag
	for _, tag := range tags {
		if tag != nil && tag.Semver == nil {
			logger.Debug("can't compute a semver from the tag => ignoring it", slog.String("tag", tag.Name))
			continue
		}
		section := &Section{
			Tag: tag,
			Prs: make([]*repo.PullRequest, 0),
		}
		for _, pr := range prs {
			if isPullRequestIncludedInThisSegment(pr, previousTag, tag, config.MinimalDelayInSeconds) {
				if pr.IsIgnored(config.PullRequestIgnoreLabels) {
					logger.Debug("ignore a PR", slog.Int("number", pr.Number))
					continue
				}
				section.Prs = append(section.Prs, pr)
			}
		}
		sections = append(sections, section)
		previousTag = tag
	}
	return &Changelog{
		RepoOwner: config.RepoOwner,
		RepoName:  config.RepoName,
		Sections:  sections,
	}
}
