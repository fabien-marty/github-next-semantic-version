package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v2"
)

func nextVersionAction(cCtx *cli.Context) error {
	setDefaultLogger(cCtx)
	branch := cCtx.String("branch")
	service, err := getService(cCtx)
	if err != nil {
		return err
	}
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
