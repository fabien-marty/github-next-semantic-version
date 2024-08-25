# github-next-semantic-version

## What is it?

`github-next-semantic-version` is a little CLI binary (written in golang) to
**guesses the next [semantic version](https://semver.org/)** from:

- existing **git tags** *(read from a locally cloned git repository)*
- and recently merged **pull-requests labels** *(read from the GitHub API)*

Unlinke plenty of "similar" tools, we don't use any "commit message parsing" here but only 
**configurable PR labels**.

Example *(with a repo cloned in the current directory)*:

```console
$ github-next-semantic-version .
v1.10.0 => v1.10.1
$ # v1.10.0 is the latest version
$ # v1.10.1 is the next version
```

> [!TIP]
> **How do we determine the next version? How do we determine if the next version is a patch/minor/major version?**
>
> - we list PRs merged since the latest tag
> - we examine corresponding PR labels:
>     - if we find at least one `breaking` or `Type: Major` label => this is a major release *(so we increment the major version number)*
>     - if we find at least one `feature` or `Type: Feature` label => this is a minor release *(so we increment the minor version number)*
>     - else this is a patch release
>
> *(of course, you can define your own labels to configure the logic)*

## Non-features

- "commit message parsing": there are plenty of tools to do that, here, we want to rely only on merged PR labels
- "other providers support": we support only "GitHub" *(feel free to fork if you want to add other providers support)*

## Installation / Quickstart

> [!NOTE]
> If you want to use this tool inside a GHA workflow, we provide a dedicated GitHub Action in
> [this dedicated repository](https://github.com/fabien-marty/github-next-semantic-version-action).

We provide compiled binaries for various architecture in the [release page](https://github.com/fabien-marty/github-next-semantic-version/releases).

- download the corresponding file
- set the "executable bit"
- clone a public repository locally (with all tags)
- execute `./github-next-semantic-version .`

> [!NOTE]
> Of course it also works with private repositories but you will need a GitHub token
> set to `GITHUB_TOKEN` env var (for example).

## CLI reference

```console
$ github-next-semantic-version --help

NAME:
   github-next-semantic-version - Compute the next semantic version with merged PRs and corresponding labels

USAGE:
   github-next-semantic-version [global options] command [command options] LOCAL_GIT_REPO_PATH

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --log-level value                 log level (DEBUG, INFO, WARN, ERROR) (default: "INFO") [$LOG_LEVEL]
   --log-format value                log format (text-human, text, json, json-gcp) (default: "text-human") [$LOG_FORMAT]
   --github-token value              github token [$GITHUB_TOKEN]
   --repo-owner value                repository owner (organization); if not set, we are going to try to guess [$GNSV_REPO_OWNER]
   --repo-name value                 repository name (without owner/organization part); if not set, we are going to try to guess [$GNSV_REPO_NAME]
   --branch value                    Branch to filter on [$GNSV_BRANCH_NAME]
   --major-labels value              Coma separated list of PR labels to consider as major (default: "major,breaking,Type: Major") [$GNSV_MAJOR_LABELS]
   --minor-labels value              Coma separated list of PR labels to consider as minor (default: "feature,Type: Feature,Type: Minor") [$GNSV_MINOR_LABELS]
   --ignore-labels value             Coma separated list of PR labels to consider as ignored PRs (default: "Type: Hidden") [$GNSV_HIDDEN_LABELS]
   --dont-increment-if-no-pr         Don't increment the version if no PR is found (or if only ignored PRs found) (default: false) [$GNSV_DONT_INCREMENT_IF_NO_PR]
   --consider-also-non-merged-prs    Consider also non-merged PRs (default: false) [$GNSV_CONSIDER_ALSO_NON_MERGED_PRS]
   --minimal-delay-in-seconds value  Minimal delay in seconds between a PR and a tag (if less, we consider that the tag is always AFTER the PR) (default: 5)
   --help, -h                        show help

```

## DEV

This tool is fully developped in Golang 1.21+ with following libraries:

- [github.com/Masterminds/semver V3](https://github.com/Masterminds/semver/): for semver parsing
- [github.com/google/go-github V62](https://github.com/google/go-github/): for GitHub API 
- [github.com/urfave/cli V2](https://github.com/urfave/cli/): for CLI

We follow [golang-standards/project-layout](https://github.com/golang-standards/project-layout) directories structure
and we use "hexagonal architecture" with:
- domain/use-cases code in the `app` subdir
- IO adapters in the `infra/adapters` subdir
- CLI controller in the `infra/controllers` subdir

Dev commands are implemented inside a `Makefile` with following targets:

```console
$ make help
build                          Build Go binaries
clean                          Clean the repo
doc                            Generate documentation
lint                           Lint the code (also fix the code if FIX=1, default)
no-dirty                       Check if the repo is dirty
test-integration               Run integration tests
test-unit                      Execute all unit tests
test                           Execute all tests 

```
