# github-next-semantic-version

[![](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=flat)](https://goreportcard.com/report/github.com/fabien-marty/github-next-semantic-version)

## What is it?

`github-next-semantic-version` is a little CLI binary (written in golang) to
**guess the next [semantic version](https://semver.org/)** from:

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

> [!NOTE]
> We also provide:
>
> - a dedicated GitHub Action in [this dedicated repository](https://github.com/fabien-marty/github-next-semantic-version-action) *if you want to use this tool inside a GHA workflow*
> - another CLI binary: `github-create-next-semantic-release` *(in this current repository)* to use the previous rules to automatically create a GitHub release with the guessed version and the corresponding release notes *(made from merged PRs and a configurable template)*
> - another GitHub Action in [this other repository](https://github.com/fabien-marty/github-create-next-semantic-release) *if you want to use this alternate tool: `github-create-next-semantic-release` inside a GHA workflow*

## Features

- support full semver specification (basic `1.2.3` but also `1.0.0-beta.2`, `0.1.9-post.24_a5256f1`...)
- can filter tags with regex (see `--tag-regex` option)
- support prefixed tags (example: `v1.2.3` but also `foo/bar/v1.2.3`...) when parsing the semantic version
- configure your own PR labels for major and minor increments
- ... (see "CLI reference" in this document)
- addon binary to automatically create GitHub releases with the guessed version and corresponding release notes

## Non-features

- "commit message parsing": there are plenty of tools to do that, here, we want to rely only on merged PR labels
- "other providers support": we support only "GitHub" *(feel free to fork if you want to add other providers support)*

## Installation / Quickstart

We provide compiled binaries for various architecture in the [release page](https://github.com/fabien-marty/github-next-semantic-version/releases).

- download the corresponding file
- set the "executable bit"
- clone a public repository locally (with all tags)
- execute `./github-next-semantic-version .`

*(same for `github-create-next-semantic-release` binary)*

> [!NOTE]
> Of course it also works with private repositories but you will need a GitHub token
> set to `GITHUB_TOKEN` env var (for example).

## CLI reference

<details open>

<summary>CLI reference of github-next-semantic-version</summary>

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
   --consider-also-non-merged-prs    Consider also non-merged PRs (default: false) [$GNSV_CONSIDER_ALSO_NON_MERGED_PRS]
   --tag-regex value                 Regex to match tags (if empty string (default) => no filtering) [$GNSV_TAG_REGEX]
   --ignore-labels value             Coma separated list of PR labels to consider as ignored PRs (default: "Type: Hidden") [$GNSV_HIDDEN_LABELS]
   --major-labels value              Coma separated list of PR labels to consider as major (default: "major,breaking,Type: Major") [$GNSV_MAJOR_LABELS]
   --minor-labels value              Coma separated list of PR labels to consider as minor (default: "feature,Type: Feature,Type: Minor") [$GNSV_MINOR_LABELS]
   --minimal-delay-in-seconds value  Minimal delay in seconds between a PR and a tag (if less, we consider that the tag is always AFTER the PR) (default: 5)
   --dont-increment-if-no-pr         Don't increment the version if no PR is found (or if only ignored PRs found) (default: false) [$GNSV_DONT_INCREMENT_IF_NO_PR]
   --next-version-only               If set, output only the next version (without the old one) (default: false) [$GNSV_NEXT_VERSION_ONLY]
   --help, -h                        show help

```

</details>

<details>

<summary>CLI reference of github-create-next-semantic-release</summary>

```console
$ github-create-next-semantic-release --help

NAME:
   github-create-next-semantic-release - Create the next semantice release on GitHub (depending on the PRs merged since the last release)

USAGE:
   github-create-next-semantic-release [global options] command [command options] LOCAL_GIT_REPO_PATH

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --log-level value                   log level (DEBUG, INFO, WARN, ERROR) (default: "INFO") [$LOG_LEVEL]
   --log-format value                  log format (text-human, text, json, json-gcp) (default: "text-human") [$LOG_FORMAT]
   --github-token value                github token [$GITHUB_TOKEN]
   --repo-owner value                  repository owner (organization); if not set, we are going to try to guess [$GNSV_REPO_OWNER]
   --repo-name value                   repository name (without owner/organization part); if not set, we are going to try to guess [$GNSV_REPO_NAME]
   --branch value                      Branch to filter on [$GNSV_BRANCH_NAME]
   --consider-also-non-merged-prs      Consider also non-merged PRs (default: false) [$GNSV_CONSIDER_ALSO_NON_MERGED_PRS]
   --tag-regex value                   Regex to match tags (if empty string (default) => no filtering) [$GNSV_TAG_REGEX]
   --ignore-labels value               Coma separated list of PR labels to consider as ignored PRs (default: "Type: Hidden") [$GNSV_HIDDEN_LABELS]
   --major-labels value                Coma separated list of PR labels to consider as major (default: "major,breaking,Type: Major") [$GNSV_MAJOR_LABELS]
   --minor-labels value                Coma separated list of PR labels to consider as minor (default: "feature,Type: Feature,Type: Minor") [$GNSV_MINOR_LABELS]
   --minimal-delay-in-seconds value    Minimal delay in seconds between a PR and a tag (if less, we consider that the tag is always AFTER the PR) (default: 5)
   --release-draft                     if set, the release is created in draft mode (default: false) [$GNSV_RELEASE_DRAFT]
   --release-body-template value       golang template to generate the release body (default: "{{ range . }}- {{.Title}} (#{{.Number}})\n{{ end }}") [$GNSV_RELEASE_BODY_TEMPLATE]
   --release-body-template-path value  golang template path to generate the release body (if set, release-body-template option is ignored) [$GNSV_RELEASE_BODY_TEMPLATE_PATH]
   --release-force                     if set, force the version bump and the creation of a release (even if there is no PR) (default: false) [$GNSV_RELEASE_FORCE]
   --help, -h                          show help

```

</details>

<details>

<summary>CLI reference of github-generate-changelog</summary>

```console
$ github-generate-changelog --help

NAME:
   github-generate-changelog - Make a changelog from local git tags and GitHub merged PRs

USAGE:
   github-generate-changelog [global options] command [command options] LOCAL_GIT_REPO_PATH

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --log-level value               log level (DEBUG, INFO, WARN, ERROR) (default: "INFO") [$LOG_LEVEL]
   --log-format value              log format (text-human, text, json, json-gcp) (default: "text-human") [$LOG_FORMAT]
   --github-token value            github token [$GITHUB_TOKEN]
   --repo-owner value              repository owner (organization); if not set, we are going to try to guess [$GNSV_REPO_OWNER]
   --repo-name value               repository name (without owner/organization part); if not set, we are going to try to guess [$GNSV_REPO_NAME]
   --branch value                  Branch to filter on [$GNSV_BRANCH_NAME]
   --consider-also-non-merged-prs  Consider also non-merged PRs (default: false) [$GNSV_CONSIDER_ALSO_NON_MERGED_PRS]
   --tag-regex value               Regex to match tags (if empty string (default) => no filtering) [$GNSV_TAG_REGEX]
   --ignore-labels value           Coma separated list of PR labels to consider as ignored PRs (default: "Type: Hidden") [$GNSV_HIDDEN_LABELS]
   --future                        if set, include a future section (default: false) [$GNSV_CHANGELOG_FUTURE]
   --template-path value           if set, define the path to the changelog template [$GNSV_CHANGELOG_TEMPLATE_PATH]
   --help, -h                      show help

```

</details>

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