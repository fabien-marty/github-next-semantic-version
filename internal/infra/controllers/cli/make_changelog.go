package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/fabien-marty/github-next-semantic-version/internal/app"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/changelog"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/git"
	gitlocal "github.com/fabien-marty/github-next-semantic-version/internal/infra/adapters/git/local"
	repogithub "github.com/fabien-marty/github-next-semantic-version/internal/infra/adapters/repo/github"
	"github.com/urfave/cli/v2"
)

func makeChangelogAction(cCtx *cli.Context) error {
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
	templateString := changelog.DefaultTemplateString
	if cCtx.String("template-path") != "" {
		templateStringBytes, err := os.ReadFile(cCtx.String("template-path"))
		if err != nil {
			return cli.Exit(fmt.Sprintf("Can't read the changelog template file: %s", err), 1)
		}
		templateString = string(templateStringBytes)
	}
	changelog, err := service.GenerateChangelog(branch, !cCtx.Bool("consider-also-non-merged-prs"), cCtx.Bool("future"), nil, templateString)
	if err != nil {
		if err == app.ErrNoRelease {
			return cli.Exit(errors.New("no need to create a release => use --release-force if you want to force a version bump and a new release"), 2)
		}
		return cli.Exit(err.Error(), 2)
	}
	fmt.Println(changelog)
	return nil
}

func MakeChangelogMain() {
	cliFlags := commonCliFlags
	cliFlags = append(cliFlags, &cli.BoolFlag{
		Name:    "future",
		Value:   false,
		Usage:   "if set, include a future section",
		EnvVars: []string{"GNSV_CHANGELOG_FUTURE"},
	})
	cliFlags = append(cliFlags, &cli.StringFlag{
		Name:    "template-path",
		Value:   "",
		Usage:   "if set, define the path to the changelog template",
		EnvVars: []string{"GNSV_CHANGELOG_TEMPLATE_PATH"},
	})
	app := &cli.App{
		Name:      "github-generate-changelog",
		Usage:     "Make a changelog from local git tags and GitHub merged PRs",
		Action:    makeChangelogAction,
		ArgsUsage: "LOCAL_GIT_REPO_PATH",
		Flags:     cliFlags,
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "bad CLI arguments: %s\n", slog.String("err", err.Error()))
		os.Exit(1)
	}
}
