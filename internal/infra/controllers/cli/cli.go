package cli

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/fabien-marty/github-next-semantic-version/internal/app"
	"github.com/fabien-marty/github-next-semantic-version/internal/infra/adapters/git/gitlocal"
	"github.com/fabien-marty/github-next-semantic-version/internal/infra/adapters/repo/repogithub"
	"github.com/fabien-marty/slog-helpers/pkg/slogc"
	"github.com/urfave/cli/v2"
)

func setDefaultLogger(cCtx *cli.Context) {
	logger := slogc.GetLogger(
		slogc.WithLevel(slogc.GetLogLevelFromString(cCtx.String("log-level"))),
		slogc.WithLogFormat(slogc.GetLogFormatFromString(cCtx.String("log-format"))),
	)
	slog.SetDefault(logger)
}

func action(cCtx *cli.Context) error {
	setDefaultLogger(cCtx)
	repoOwner := cCtx.String("repo-owner")
	repoName := cCtx.String("repo-name")
	branch := cCtx.String("branch")
	repoGithubAdapter := repogithub.NewAdapter(repoOwner, repoName, repogithub.AdapterOptions{Token: cCtx.String("github-token")})
	gitLocalAdapter := gitlocal.NewAdapter(gitlocal.AdapterOptions{
		LocalGitPath: cCtx.String("git-repository-local-path"),
	})
	appConfig := app.Config{
		PullRequestMajorLabels: strings.Split(cCtx.String("major-labels"), ","),
		PullRequestMinorLabels: strings.Split(cCtx.String("minor-labels"), ","),
	}
	service := app.NewService(appConfig, repoGithubAdapter, gitLocalAdapter)
	res, err := service.GetNextVersion(branch, !cCtx.Bool("consider-also-non-merged-prs"))
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}
	fmt.Println(res)
	return nil
}

func Main() {
	app := &cli.App{
		Name:   "github-next-semantic-version",
		Usage:  "Compute the next semantic version with merged PRs and corresponding labels",
		Action: action,
		Flags: []cli.Flag{
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
				Name:     "repo-owner",
				Required: true,
				Usage:    "repository owner (organization)",
				EnvVars:  []string{"REPO_OWNER"},
			},
			&cli.StringFlag{
				Name:     "repo-name",
				Required: true,
				Usage:    "repository name (without owner/organization part)",
				EnvVars:  []string{"REPO_NAME"},
			},
			&cli.StringFlag{
				Name:    "branch",
				Value:   "main",
				Usage:   "Branch to filter on (probably the main branch)",
				EnvVars: []string{"BRANCH_NAME"},
			},
			&cli.StringFlag{
				Name:    "major-labels",
				Value:   "major,breaking,Type: Major",
				Usage:   "Coma separated list of PR labels to consider as major",
				EnvVars: []string{"MAJOR_LABELS"},
			},
			&cli.StringFlag{
				Name:    "minor-labels",
				Value:   "feature,Type: Feature,Type: Minor",
				Usage:   "Coma separated list of PR labels to consider as minor",
				EnvVars: []string{"MINOR_LABELS"},
			},
			&cli.StringFlag{
				Name:  "git-repository-local-path",
				Value: ".",
				Usage: "Git repository local path",
			},
			&cli.BoolFlag{
				Name:  "consider-also-non-merged-prs",
				Value: false,
				Usage: "Consider also non-merged PRs",
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "bad CLI arguments: %s", slog.String("err", err.Error()))
		os.Exit(1)
	}
}
