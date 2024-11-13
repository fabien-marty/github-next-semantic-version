package git

// Port is the interface that must be implemented by git adapters.
type Port interface {
	// GetContainedTags returns the list of tags contained by the given branch.
	GetContainedTags(branch string) ([]*Tag, error)
	GuessGHRepo() (owner string, repo string)
	GuessDefaultBranch() string
}
