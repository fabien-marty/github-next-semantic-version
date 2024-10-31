package cli

import (
	"log/slog"
	"os"

	"github.com/fabien-marty/slog-helpers/pkg/slogc"
	"github.com/urfave/cli/v2"
)

var commonCliFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "log-level",
		Value:   "INFO",
		Usage:   "log level (DEBUG, INFO, WARN, ERROR)",
		EnvVars: []string{"LOG_LEVEL"},
	},
	&cli.StringFlag{
		Name:    "log-format",
		Value:   "text-human",
		Usage:   "log format (text-human, text, json, json-gcp)",
		EnvVars: []string{"LOG_FORMAT"},
	},
	&cli.StringFlag{
		Name:    "github-token",
		Usage:   "github token",
		EnvVars: []string{"GITHUB_TOKEN"},
	},
	&cli.StringFlag{
		Name:    "repo-owner",
		Usage:   "repository owner (organization); if not set, we are going to try to guess",
		EnvVars: []string{"GNSV_REPO_OWNER"},
	},
	&cli.StringFlag{
		Name:    "repo-name",
		Usage:   "repository name (without owner/organization part); if not set, we are going to try to guess",
		EnvVars: []string{"GNSV_REPO_NAME"},
	},
	&cli.StringFlag{
		Name:    "branch",
		Value:   "",
		Usage:   "Branch to filter on",
		EnvVars: []string{"GNSV_BRANCH_NAME"},
	},
	&cli.StringFlag{
		Name:    "major-labels",
		Value:   "major,breaking,Type: Major",
		Usage:   "Coma separated list of PR labels to consider as major",
		EnvVars: []string{"GNSV_MAJOR_LABELS"},
	},
	&cli.StringFlag{
		Name:    "minor-labels",
		Value:   "feature,Type: Feature,Type: Minor",
		Usage:   "Coma separated list of PR labels to consider as minor",
		EnvVars: []string{"GNSV_MINOR_LABELS"},
	},
	&cli.StringFlag{
		Name:    "ignore-labels",
		Value:   "Type: Hidden",
		Usage:   "Coma separated list of PR labels to consider as ignored PRs",
		EnvVars: []string{"GNSV_HIDDEN_LABELS"},
	},
	&cli.BoolFlag{
		Name:    "consider-also-non-merged-prs",
		Value:   false,
		Usage:   "Consider also non-merged PRs",
		EnvVars: []string{"GNSV_CONSIDER_ALSO_NON_MERGED_PRS"},
	},
	&cli.IntFlag{
		Name:  "minimal-delay-in-seconds",
		Value: 5,
		Usage: "Minimal delay in seconds between a PR and a tag (if less, we consider that the tag is always AFTER the PR)",
	},
	&cli.StringFlag{
		Name:    "tag-regex",
		Value:   "",
		Usage:   "Regex to match tags (if empty string (default) => no filtering)",
		EnvVars: []string{"GNSV_TAG_REGEX"},
	},
}

func setDefaultLogger(cCtx *cli.Context) {
	logger := slogc.GetLogger(
		slogc.WithLevel(slogc.GetLogLevelFromString(cCtx.String("log-level"))),
		slogc.WithLogFormat(slogc.GetLogFormatFromString(cCtx.String("log-format"))),
	)
	slog.SetDefault(logger)
}

func guessGHRepoFromEnv() (owner string, repo string) {
	ghOwner := os.Getenv("GITHUB_REPOSITORY_OWNER")
	ghRepository := os.Getenv("GITHUB_REPOSITORY")
	if ghOwner != "" && ghRepository != "" {
		// we are in a GitHub Actions environment
		return ghOwner, ghRepository[len(ghOwner)+1:]
	}
	return "", ""
}