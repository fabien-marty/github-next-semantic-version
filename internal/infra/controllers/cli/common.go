package cli

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/fabien-marty/github-next-semantic-version/internal/app"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/git"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/repo"
	gitlocal "github.com/fabien-marty/github-next-semantic-version/internal/infra/adapters/git/local"
	repocache "github.com/fabien-marty/github-next-semantic-version/internal/infra/adapters/repo/cache"
	repogithub "github.com/fabien-marty/github-next-semantic-version/internal/infra/adapters/repo/github"
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
		Name:    "branches",
		Aliases: []string{"branch"},
		Value:   "",
		Usage:   "Coma separated list of branch names to filter on for getting tags and prs (if not set, the default branch is guessed/used)",
		EnvVars: []string{"GNSV_BRANCH_NAME"},
	},
	&cli.BoolFlag{
		Name:    "consider-also-non-merged-prs",
		Value:   false,
		Usage:   "Consider also non-merged PRs",
		EnvVars: []string{"GNSV_CONSIDER_ALSO_NON_MERGED_PRS"},
	},
	&cli.StringFlag{
		Name:    "tag-regex",
		Value:   "",
		Usage:   "Regex to match tags (if empty string (default) => no filtering)",
		EnvVars: []string{"GNSV_TAG_REGEX"},
	},
	&cli.StringFlag{
		Name:    "ignore-labels",
		Value:   "Type: Hidden",
		Usage:   "Coma separated list of PR labels to consider as ignored PRs (OR condition)",
		EnvVars: []string{"GNSV_HIDDEN_LABELS"},
	},
	&cli.StringFlag{
		Name:    "must-have-labels",
		Value:   "",
		Usage:   "Coma separated list of PR labels that PRs must have to be considered (OR condition, empty => no filtering)",
		EnvVars: []string{"GNSV_MUST_HAVE_LABELS"},
	},
	&cli.IntFlag{
		Name:  "minimal-delay-in-seconds",
		Value: 5,
		Usage: "Minimal delay in seconds between a PR and a tag (if less, we consider that the tag is always AFTER the PR)",
	},
	&cli.BoolFlag{
		Name:    "cache",
		Value:   false,
		Usage:   "Cache pull-requests read",
		EnvVars: []string{"GNSV_CACHE"},
	},
	&cli.IntFlag{
		Name:    "cache-lifetime",
		Value:   3600,
		Usage:   "Lifetime (in seconds) of the pull-requests cache",
		EnvVars: []string{"GNSV_CACHE_LIFETIME"},
	},
	&cli.StringFlag{
		Name:    "cache-location",
		Value:   ".",
		Usage:   "Cache Location (directory that must exist)",
		EnvVars: []string{"GNSV_CACHE_LOCATION"},
	},
	&cli.BoolFlag{
		Name:    "cache-dont-try-to-update",
		Value:   false,
		Usage:   "If set, don't try to update the cache (use it only if you know what you are doing)",
		EnvVars: []string{"GNSV_CACHE_DONT_TRY_TO_UPDATE"},
	},
}

func addExtraCommonCliFlags(cliFlags []cli.Flag) []cli.Flag {
	res := make([]cli.Flag, len(cliFlags))
	copy(res, cliFlags)
	res = append(res, &cli.StringFlag{
		Name:    "major-labels",
		Value:   "major,breaking,Type: Major,Type: Breaking",
		Usage:   "Coma separated list of PR labels to consider as major (OR condition)",
		EnvVars: []string{"GNSV_MAJOR_LABELS"},
	})
	res = append(res, &cli.StringFlag{
		Name:    "minor-labels",
		Value:   "feature,Type: Feature,Type: Minor,Type: Added",
		Usage:   "Coma separated list of PR labels to consider as minor (OR condition)",
		EnvVars: []string{"GNSV_MINOR_LABELS"},
	})
	return res
}

func setDefaultLogger(cCtx *cli.Context) {
	logger := slogc.GetLogger(
		slogc.WithLevel(slogc.GetLogLevelFromString(cCtx.String("log-level"))),
		slogc.WithLogFormat(slogc.GetLogFormatFromString(cCtx.String("log-format"))),
	)
	slog.SetDefault(logger)
}

func getRepoOwnerAndRepoName(cCtx *cli.Context, gitLocalAdapter git.Port) (repoOwner string, repoName string, err error) {
	repoOwner = cCtx.String("repo-owner")
	repoName = cCtx.String("repo-name")
	if repoOwner == "" || repoName == "" {
		ghActions := os.Getenv("GITHUB_ACTIONS")
		if ghActions == "true" {
			repoOwner, repoName = guessGHRepoFromEnv()
		} else {
			repoOwner, repoName = gitLocalAdapter.GuessGHRepo()
		}
		if repoOwner == "" || repoName == "" {
			return "", "", cli.Exit("Can't guess the repository owner and name => please provide them as CLI flags", 1)
		}
	}
	return repoOwner, repoName, nil
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

func specialSplit(s string, sep string) []string {
	res := []string{}
	if s == "" {
		return []string{}
	}
	tmp := strings.Split(s, sep)
	for _, t := range tmp {
		trimT := strings.TrimSpace(t)
		if trimT != "" {
			res = append(res, trimT)
		}
	}
	return res
}

func getService(cCtx *cli.Context) (*app.Service, error) {
	localGitPath := cCtx.Args().Get(0)
	if localGitPath == "" {
		return nil, cli.Exit("You have to set LOCAL_GIT_REPO_PATH argument (use . for the currently dir)", 1)
	}
	var gitLocalAdapter git.Port = gitlocal.NewAdapter(gitlocal.AdapterOptions{
		LocalGitPath: localGitPath,
	})
	repoOwner, repoName, err := getRepoOwnerAndRepoName(cCtx, gitLocalAdapter)
	if err != nil {
		return nil, err
	}
	slog.Debug(fmt.Sprintf("Repository owner: %s, repository name: %s", repoOwner, repoName))
	repoGithubAdapter := repogithub.NewAdapter(repoOwner, repoName, repogithub.AdapterOptions{Token: cCtx.String("github-token")})
	var repoAdapter repo.Port = repoGithubAdapter
	if cCtx.Bool("cache") {
		repoAdapter = repocache.NewAdapter(repoOwner, repoName, repoGithubAdapter, repocache.AdapterOptions{
			CacheLocation:        cCtx.String("cache-location"),
			CacheLifetime:        cCtx.Int("cache-lifetime"),
			CacheDontTryToUpdate: cCtx.Bool("cache-dont-try-to-update"),
		})
	}
	appConfig := app.Config{
		PullRequestMajorLabels:    specialSplit(cCtx.String("major-labels"), ","),
		PullRequestMinorLabels:    specialSplit(cCtx.String("minor-labels"), ","),
		PullRequestIgnoreLabels:   specialSplit(cCtx.String("ignore-labels"), ","),
		PullRequestMustHaveLabels: specialSplit(cCtx.String("must-have-labels"), ","),
		MinimalDelayInSeconds:     cCtx.Int("minimal-delay-in-seconds"),
		TagRegex:                  cCtx.String("tag-regex"),
		RepoOwner:                 repoOwner,
		RepoName:                  repoName,
	}
	service := app.NewService(appConfig, repoAdapter, gitLocalAdapter)
	return service, nil
}

func getBranches(cCtx *cli.Context, service *app.Service) []string {
	branches := specialSplit(cCtx.String("branches"), ",")
	if len(branches) == 0 {
		branch := service.GitAdapter.GuessDefaultBranch()
		slog.Debug(fmt.Sprintf("branch guessed: %s", branch))
		branches = append(branches, branch)
	}
	return branches
}
