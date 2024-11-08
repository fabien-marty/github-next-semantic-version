package app

// Config is the configuration of the application
type Config struct {
	RepoOwner               string   // Repository owner name (organization)
	RepoName                string   // Repository name (without owner/organization part)
	PullRequestMajorLabels  []string // list of labels for considering a PR as major
	PullRequestMinorLabels  []string // list of labels for considering a PR as minor
	PullRequestIgnoreLabels []string // list of labels for completely ignoring a PR
	MinimalDelayInSeconds   int      // minimal delay in seconds between a PR and a tag (if less, we consider that the tag is always AFTER the PR)
	TagRegex                string   // regex to match tags (if empty string => no filtering)
}
