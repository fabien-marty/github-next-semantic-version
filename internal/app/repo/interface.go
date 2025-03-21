package repo

// Port is the interface that must be implemented by repo adapters.
type Port interface {

	// GetPullRequestsSince returns the list of pull requests (targetting the given base)
	// If onlyMerged is true, only the merged pull requests
	// If onlyMerged is false, merged pull requests + (still) open pull requests.
	GetPullRequests(base string, onlyMerged bool) ([]*PullRequest, error)

	// GetLastUpdatedPullRequests returns the first page of pull requests (targetting the given base)
	// sorted by "last updated". This method doesn't paginate so you won't get all the pull requests
	// (only the first page).
	// If onlyMerged is true, only the merged pull requests
	// If onlyMerged is false, merged pull requests + (still) open pull requests.
	// The list is sorted by updatedAt (descending).
	GetLastUpdatedPullRequests(base string, onlyMerged bool) ([]*PullRequest, error)

	CreateRelease(base string, tagName string, body string, draft bool) error
}
