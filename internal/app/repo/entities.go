package repo

import "time"

// PullRequestConfig holds labels to identify major and minor pull requests.
type PullRequestConfig struct {
	MajorLabels []string
	MinorLabels []string
}

// PullRequest represents a pull request.
type PullRequest struct {
	Number   int        // pull request number
	Title    string     // pull request title
	MergedAt *time.Time // pull request merge date (nil if not merged)
	Labels   []string   // pull request labels
}

// IsMajor returns true if the pull request is a major one.
// A pull request is considered major if it has at least one of the major labels.
func (pr *PullRequest) IsMajor(config PullRequestConfig) bool {
	for _, label := range pr.Labels {
		for _, majorLabel := range config.MajorLabels {
			if label == majorLabel {
				return true
			}
		}
	}
	return false
}

// IsMinor returns true if the pull request is a minor one.
// A pull request is considered minor if it has at least one of the minor labels.
func (pr *PullRequest) IsMinor(config PullRequestConfig) bool {
	for _, label := range pr.Labels {
		for _, minorLabel := range config.MinorLabels {
			if label == minorLabel {
				return true
			}
		}
	}
	return false
}

// IsPatch returns true if the pull request is a patch one.
// A pull request is considered patch if it is neither major nor minor.
func (pr *PullRequest) IsPatch(config PullRequestConfig) bool {
	return !pr.IsMajor(config) && !pr.IsMinor(config)
}

// IsMerged returns true if the pull request is merged.
func (pr *PullRequest) IsMerged() bool {
	return pr.MergedAt != nil
}
