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

func nextVersionAction(cCtx *cli.Context) error {
	setDefaultLogger(cCtx)
	localGitPath := cCtx.Args().Get(0)
	if localGitPath == "" {
		return cli.Exit("You have to set LOCAL_GIT_REPO_PATH argument (use . for the currently dir)", 1)
	}
	var gitLocalAdapter git.Port = gitlocal.NewAdapter(gitlocal.AdapterOptions{
		LocalGitPath: localGitPath,
	})
	repoOwner, repoName, err := getRepoOwnerAndRepoName(cCtx, gitLocalAdapter)
	if err != nil {
		return err
	}
	slog.Debug(fmt.Sprintf("Repository owner: %s, repository name: %s", repoOwner, repoName))
	branch := cCtx.String("branch")
	repoGithubAdapter := repogithub.NewAdapter(repoOwner, repoName, repogithub.AdapterOptions{Token: cCtx.String("github-token")})
	appConfig := app.Config{
		PullRequestMajorLabels:  strings.Split(cCtx.String("major-labels"), ","),
		PullRequestMinorLabels:  strings.Split(cCtx.String("minor-labels"), ","),
		PullRequestIgnoreLabels: strings.Split(cCtx.String("ignore-labels"), ","),
		MinimalDelayInSeconds:   cCtx.Int("minimal-delay-in-seconds"),
		TagRegex:                cCtx.String("tag-regex"),
		RepoOwner:               repoOwner,
		RepoName:                repoName,
	}
	service := app.NewService(appConfig, repoGithubAdapter, gitLocalAdapter)
	oldVersion, newVersion, _, err := service.GetNextVersion(branch, !cCtx.Bool("consider-also-non-merged-prs"), cCtx.Bool("dont-increment-if-no-pr"))
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}
	if cCtx.Bool("next-version-only") {
		fmt.Printf("%s\n", newVersion)
	} else {
		fmt.Printf("%s => %s\n", oldVersion, newVersion)
	}
	return nil
}

func NextVersionMain() {
	cliFlags := addExtraCommonCliFlags(commonCliFlags)
	cliFlags = append(cliFlags, &cli.BoolFlag{
		Name:    "dont-increment-if-no-pr",
		Value:   false,
		Usage:   "Don't increment the version if no PR is found (or if only ignored PRs found)",
		EnvVars: []string{"GNSV_DONT_INCREMENT_IF_NO_PR"},
	})
	cliFlags = append(cliFlags, &cli.BoolFlag{
		Name:    "next-version-only",
		Value:   false,
		Usage:   "If set, output only the next version (without the old one)",
		EnvVars: []string{"GNSV_NEXT_VERSION_ONLY"},
	})
	app := &cli.App{
		Name:      "github-next-semantic-version",
		Usage:     "Compute the next semantic version with merged PRs and corresponding labels",
		Action:    nextVersionAction,
		ArgsUsage: "LOCAL_GIT_REPO_PATH",
		Flags:     cliFlags,
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "bad CLI arguments: %s\n", slog.String("err", err.Error()))
		os.Exit(1)
	}
}
