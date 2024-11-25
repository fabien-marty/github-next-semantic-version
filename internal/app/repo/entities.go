package repo

import (
	"time"
)

// PullRequest represents a pull request.
type PullRequest struct {
	Number      int        // pull request number
	Title       string     // pull request title
	MergedAt    *time.Time // pull request merge date (nil if not merged)
	Labels      []string   // pull request labels
	Branch      string     // pull request branch
	Url         string     // pull request url
	AuthorLogin string     // pull request author login
	AuthorUrl   string     // pull request author url
}

// HasThisLabel returns true if the pull request has the given label
func (pr *PullRequest) HasThisLabel(label string) bool {
	for _, l := range pr.Labels {
		if l == label {
			return true
		}
	}
	return false
}

// HasOneOfTheseLabels returns true if the pull request has at least one of the given labels.
func (pr *PullRequest) HasOneOfTheseLabels(labels []string) bool {
	if pr.Labels == nil {
		return false
	}
	for _, label := range pr.Labels {
		for _, l := range labels {
			if l == label {
				return true
			}
		}
	}
	return false
}

// IsMajor returns true if the pull request is a major one.
// A pull request is considered major if it has at least one of the major labels.
func (pr *PullRequest) IsMajor(majorLabels []string) bool {
	return pr.HasOneOfTheseLabels(majorLabels)
}

// IsMinor returns true if the pull request is a minor one.
// A pull request is considered minor if it has at least one of the minor labels.
func (pr *PullRequest) IsMinor(minorLabels []string) bool {
	return pr.HasOneOfTheseLabels(minorLabels)
}

// IsIgnored returns true if the pull request is ignored.
// A pull request is considered ignored if it has at least one of the ignored labels.
func (pr *PullRequest) IsIgnored(ignoredLabels []string) bool {
	return pr.HasOneOfTheseLabels(ignoredLabels)
}

// IsPatch returns true if the pull request is a patch one.
// A pull request is considered patch if it is neither major nor minor.
func (pr *PullRequest) IsPatch(majorLabels []string, minorLabels []string, ignoredLabels []string) bool {
	return !pr.IsMajor(majorLabels) && !pr.IsMinor(minorLabels) && !pr.IsIgnored(ignoredLabels)
}

// IsMerged returns true if the pull request is merged.
func (pr *PullRequest) IsMerged() bool {
	return pr.MergedAt != nil
}
