package repo

import "time"

// Port is the interface that must be implemented by repo adapters.
type Port interface {
	// GetPullRequestsSince returns the list of pull requests since the given date.
	// If onlyMerged is true, only the merged pull requests since the given date are returned.
	// If onlyMerged is false, merged pull requests since the given date are returned + (still) open pull requests.
	GetPullRequestsSince(base string, t time.Time, onlyMerged bool) ([]PullRequest, error)
	CreateRelease(base string, tagName string, body string, draft bool) error
}
