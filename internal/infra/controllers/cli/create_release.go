package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/fabien-marty/github-next-semantic-version/internal/app"
	"github.com/urfave/cli/v2"
)

func createReleaseAction(cCtx *cli.Context) error {
	setDefaultLogger(cCtx)
	service, err := getService(cCtx)
	if err != nil {
		return err
	}
	branches := getBranches(cCtx, service)
	releaseBodyTemplate := cCtx.String("release-body-template")
	if cCtx.String("release-body-template-path") != "" {
		body, err := os.ReadFile(cCtx.String("release-body-template-path"))
		if err != nil {
			return cli.Exit(fmt.Sprintf("Can't read the release body template file: %s", err), 1)
		}
		releaseBodyTemplate = string(body)
	}
	newTag, err := service.CreateNextRelease(branches, !cCtx.Bool("release-force"), cCtx.Bool("release-draft"), releaseBodyTemplate)
	if err != nil {
		if err == app.ErrNoRelease {
			return cli.Exit(errors.New("no need to create a release => use --release-force if you want to force a version bump and a new release"), 2)
		}
		return cli.Exit(err.Error(), 2)
	}
	fmt.Println(newTag)
	return nil
}

func CreateReleaseMain() {
	cliFlags := addExtraCommonCliFlags(commonCliFlags)
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
