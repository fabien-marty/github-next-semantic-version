package app

import "github.com/fabien-marty/github-next-semantic-version/internal/app/repo"

// Config is the configuration of the application
type Config struct {
	PullRequestMajorLabels  []string // list of labels for considering a PR as major
	PullRequestMinorLabels  []string // list of labels for considering a PR as minor
	PullRequestIgnoreLabels []string // list of labels for completely ignoring a PR
	DontIncrementIfNoPR     bool     // if true, the version will not be incremented if no PR is found (or if all PRs are ignored)
	MinimalDelayInSeconds   int      // minimal delay in seconds between a PR and a tag (if less, we consider that the tag is always AFTER the PR)
}

// PullRequestConfig returns the configuration object for repo package
func (c *Config) PullRequestConfig() repo.PullRequestConfig {
	return repo.PullRequestConfig{
		MajorLabels:   c.PullRequestMajorLabels,
		MinorLabels:   c.PullRequestMinorLabels,
		IgnoredLabels: c.PullRequestIgnoreLabels,
	}
}
