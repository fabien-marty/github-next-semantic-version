package app

import "github.com/fabien-marty/github-next-semantic-version/internal/app/repo"

// Config is the configuration of the application
type Config struct {
	PullRequestMajorLabels []string // list of labels that are considered as major
	PullRequestMinorLabels []string // list of labels that are considered as minor
}

// PullRequestConfig returns the configuration object for repo package
func (c *Config) PullRequestConfig() repo.PullRequestConfig {
	return repo.PullRequestConfig{
		MajorLabels: c.PullRequestMajorLabels,
		MinorLabels: c.PullRequestMinorLabels,
	}
}
