package cli

import (
	"errors"
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

func createReleaseAction(cCtx *cli.Context) error {
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
		MinimalDelayInSeconds:   cCtx.Int("minimal-delay-in-seconds"),
		TagRegex:                cCtx.String("tag-regex"),
	}
	service := app.NewService(appConfig, repoGithubAdapter, gitLocalAdapter)
	releaseBodyTemplate := cCtx.String("release-body-template")
	if cCtx.String("release-body-template-path") != "" {
		body, err := os.ReadFile(cCtx.String("release-body-template-path"))
		if err != nil {
			return cli.Exit(fmt.Sprintf("Can't read the release body template file: %s", err), 1)
		}
		releaseBodyTemplate = string(body)
	}
	newTag, err := service.CreateNextRelease(branch, !cCtx.Bool("release-force"), cCtx.Bool("release-draft"), releaseBodyTemplate)
	if err != nil {
		if err == app.ErrNoRelease {
			return cli.Exit(errors.New("no need to create a release => use --release-force if you want to force a version bump and a new release"), 2)
		}
		return cli.Exit(err.Error(), 1)
	}
	fmt.Println(newTag)
	return nil
}

func CreateReleaseMain() {
	cliFlags := commonCliFlags
	cliFlags = append(cliFlags, &cli.BoolFlag{
		Name:    "release-draft",
		Value:   false,
		Usage:   "if set, the release is created in draft mode",
		EnvVars: []string{"GNSV_RELEASE_DRAFT"},
	})
	cliFlags = append(cliFlags, &cli.StringFlag{
		Name:    "release-body-template",
		Value:   "{{ range . }}- {{.Title}} (#{{.Number}})\n{{ end }}",
		Usage:   "golang template to generate the release body",
		EnvVars: []string{"GNSV_RELEASE_BODY_TEMPLATE"},
	})
	cliFlags = append(cliFlags, &cli.StringFlag{
		Name:    "release-body-template-path",
		Value:   "",
		Usage:   "golang template path to generate the release body (if set, release-body-template option is ignored)",
		EnvVars: []string{"GNSV_RELEASE_BODY_TEMPLATE_PATH"},
	})
	cliFlags = append(cliFlags, &cli.BoolFlag{
		Name:    "release-force",
		Usage:   "if set, force the version bump and the creation of a release (even if there is no PR)",
		EnvVars: []string{"GNSV_RELEASE_FORCE"},
	})
	app := &cli.App{
		Name:      "github-create-next-semantic-release",
		Usage:     "Create the next semantice release on GitHub (depending on the PRs merged since the last release)",
		Action:    createReleaseAction,
		ArgsUsage: "LOCAL_GIT_REPO_PATH",
		Flags:     cliFlags,
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "bad CLI arguments: %s\n", slog.String("err", err.Error()))
		os.Exit(1)
	}
}
