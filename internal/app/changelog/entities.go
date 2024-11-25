package changelog

import (
	"slices"
	"time"

	"github.com/fabien-marty/github-next-semantic-version/internal/app/git"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/repo"
)

type Config struct {
	MinimalDelayInSeconds int
	Future                bool
	RepoOwner             string
	RepoName              string
	HideMainHeader        bool
	HideSectionHeaders    bool
}

type Section struct {
	Tag *git.Tag            // Tag (or nil if this is the "future" section)
	Prs []*repo.PullRequest // List of pull-requests "contained" by this tag
}

type Changelog struct {
	Sections           []*Section
	RepoOwner          string // Repository owner name (organization)
	RepoName           string // Repository name (without owner/organization part)
	HideMainHeader     bool   // If true, don't display main header
	HideSectionHeaders bool   // If true, don't display section headers
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
	if tag2 == nil || tag2.Time == nil { //"future" tag
		return true
	}
	if tag2.Time.Add(time.Second * time.Duration(minimalDelayInSeconds)).Before(*pr.MergedAt) {
		return false
	}
	return true
}

// New creates a new Changelog instance with the given tags and pull-requests
// tags must be sorted by semver (ascending)
// prs must be sorted by mergedAt (ascending)
func New(tags []*git.Tag, prs []*repo.PullRequest, config Config) *Changelog {
	sections := []*Section{}
	if config.Future {
		tags = append(tags, nil)
	}
	var previousTag *git.Tag
	for _, tag := range tags {
		section := &Section{
			Tag: tag,
			Prs: make([]*repo.PullRequest, 0),
		}
		for _, pr := range prs {
			if isPullRequestIncludedInThisSegment(pr, previousTag, tag, config.MinimalDelayInSeconds) {
				section.Prs = append(section.Prs, pr)
			}
		}
		sections = append(sections, section)
		previousTag = tag
	}
	return &Changelog{
		RepoOwner:          config.RepoOwner,
		RepoName:           config.RepoName,
		Sections:           sections,
		HideMainHeader:     config.HideMainHeader,
		HideSectionHeaders: config.HideSectionHeaders,
	}
}
