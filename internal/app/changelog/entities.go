package changelog

import (
	"sort"
	"time"

	"github.com/fabien-marty/github-next-semantic-version/internal/app/git"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/repo"
)

type ChangelogSection struct {
	Tag *git.Tag            // Tag (or nil if this is the "future" section)
	Prs []*repo.PullRequest // List of pull-requests "contained" by this tag
}

type Changelog struct {
	Sections []*ChangelogSection
}

func (c *Changelog) GetFuturePrs() []*repo.PullRequest {
	for _, section := range c.Sections {
		if section.Tag == nil {
			return section.Prs
		}
	}
	return nil
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

func New(tags []*git.Tag, prs []*repo.PullRequest, minimalDelayInSeconds int) *Changelog {
	sections := []*ChangelogSection{}
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Time.Before(tags[j].Time)
	})
	tags = append(tags, nil)
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
		section := &ChangelogSection{
			Tag: tag,
			Prs: make([]*repo.PullRequest, 0),
		}
		for _, pr := range prs {
			if isPullRequestIncludedInThisSegment(pr, previousTag, tag, minimalDelayInSeconds) {
				section.Prs = append(section.Prs, pr)
			}
		}
		sections = append(sections, section)
		previousTag = tag
	}
	return &Changelog{Sections: sections}
}
