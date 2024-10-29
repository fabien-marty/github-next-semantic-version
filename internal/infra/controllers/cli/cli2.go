package cli

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/fabien-marty/github-next-semantic-version/internal/app"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/git"
	gitlocal "github.com/fabien-marty/github-next-semantic-version/internal/infra/adapters/git/local"
	repogithub "github.com/fabien-marty/github-next-semantic-version/internal/infra/adapters/repo/github"
	"github.com/urfave/cli/v2"
)

func action2(cCtx *cli.Context) error {
	setDefaultLogger(cCtx)
	localGitPath := cCtx.Args().Get(0)
	if localGitPath == "" {
		return cli.Exit("You have to set LOCAL_GIT_REPO_PATH argument (use . for the currently dir)", 1)
	}
	var gitLocalAdapter git.Port = gitlocal.NewAdapter(gitlocal.AdapterOptions{
		LocalGitPath: localGitPath,
	})
	repoOwner := cCtx.String("repo-owner")
	repoName := cCtx.String("repo-name")
	if repoOwner == "" || repoName == "" {
		ghActions := os.Getenv("GITHUB_ACTIONS")
		if ghActions == "true" {
			repoOwner, repoName = guessGHRepoFromEnv()
		} else {
			repoOwner, repoName = gitLocalAdapter.GuessGHRepo()
		}
		if repoOwner == "" || repoName == "" {
			return cli.Exit("Can't guess the repository owner and name => please provide them as CLI flags", 1)
		}
	}
	slog.Debug(fmt.Sprintf("Repository owner: %s, repository name: %s", repoOwner, repoName))
	branch := cCtx.String("branch")
	repoGithubAdapter := repogithub.NewAdapter(repoOwner, repoName, repogithub.AdapterOptions{Token: cCtx.String("github-token")})
	appConfig := app.Config{
		PullRequestMajorLabels:  strings.Split(cCtx.String("major-labels"), ","),
		PullRequestMinorLabels:  strings.Split(cCtx.String("minor-labels"), ","),
		PullRequestIgnoreLabels: strings.Split(cCtx.String("ignore-labels"), ","),
		DontIncrementIfNoPR:     cCtx.Bool("dont-increment-if-no-pr"),
		MinimalDelayInSeconds:   cCtx.Int("minimal-delay-in-seconds"),
		TagRegex:                cCtx.String("tag-regex"),
	}
	service := app.NewService(appConfig, repoGithubAdapter, gitLocalAdapter)
	err := service.CreateNextRelease(branch, false, cCtx.Bool("release-draft"), cCtx.String("release-body-template"))
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}
	fmt.Printf("OK")
	return nil
}

func Main2() {
	app := &cli.App{
		Name:      "github-next-semantic-version",
		Usage:     "Compute the next semantic version with merged PRs and corresponding labels",
		Action:    action2,
		ArgsUsage: "LOCAL_GIT_REPO_PATH",
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
				Name:    "dont-increment-if-no-pr",
				Value:   false,
				Usage:   "Don't increment the version if no PR is found (or if only ignored PRs found)",
				EnvVars: []string{"GNSV_DONT_INCREMENT_IF_NO_PR"},
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
			&cli.BoolFlag{
				Name:    "release-draft",
				Value:   false,
				Usage:   "if set, the release is created in draft mode",
				EnvVars: []string{"GNSV_RELEASE_DRAFT"},
			},
			&cli.StringFlag{
				Name:    "release-body-template",
				Value:   "- {{.Title}} (#{{.Number}})\n",
				Usage:   "golang template to generate the release body",
				EnvVars: []string{"GNSV_RELEASE_BODY_TEMPLATE"},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "bad CLI arguments: %s\n", slog.String("err", err.Error()))
		os.Exit(1)
	}
}