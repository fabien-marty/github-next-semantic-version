package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/fabien-marty/github-next-semantic-version/internal/app"
	"github.com/urfave/cli/v2"
)

func makeChangelogAction(cCtx *cli.Context) error {
	setDefaultLogger(cCtx)
	service, err := getService(cCtx)
	if err != nil {
		return err
	}
	branches := getBranches(cCtx, service)
	templateString, err := getTemplateString(cCtx)
	if err != nil {
		return err
	}
	if cCtx.String("starting-tag") == "LATEST" && !cCtx.Bool("future") {
		return cli.Exit("LATEST is only compatible with --future", 1)
	}
	changelog, err := service.GenerateChangelog(branches, !cCtx.Bool("consider-also-non-merged-prs"), cCtx.Bool("future"), cCtx.String("starting-tag"), templateString, cCtx.Bool("ignore-preleases"))
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
	cliFlags = append(cliFlags, &cli.StringFlag{
		Name:    "starting-tag",
		Value:   "",
		Usage:   "if set, defining a starting tag (excluded) for changelog generation, the special value 'LATEST' (combined with --future) will use the latest semantic tag to get only the future section",
		EnvVars: []string{"GNSV_CHANGELOG_STARTING_TAG"},
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
