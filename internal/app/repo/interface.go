package repo

// Port is the interface that must be implemented by repo adapters.
type Port interface {
	// GetPullRequestsSince returns the list of pull requests
	// If onlyMerged is true, only the merged pull requests
	// If onlyMerged is false, merged pull requests + (still) open pull requests.
	GetPullRequestsSince(base string, onlyMerged bool) ([]*PullRequest, error)
	CreateRelease(base string, tagName string, body string, draft bool) error
}
