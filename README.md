# github-next-semantic-version

## What is it?

`github-next-semantic-version` is a little CLI binary (written in golang) to **guess the next semantic version** from:

- existing **git tags** *(read from a locally cloned git repository)*
- and recently merged **pull-requests labels** *(read from the GitHub API)*

Unlinke plenty of "similar" tools, we don't use any "commit message parsing" here but only configurable PR labels.

Example *(with a repo cloned in the current directory)*:

```console
$ github-next-semantic-version .
v1.10.0 => v1.10.1
```

## Usage with GitHub Actions

By default, `checkout` GHA does not fetch tags. You have to use something like that:

```yaml
- uses: actions/checkout@v4
  with:
    fetch-tags: true
    fetch-depth: 0 # fetch-tags is not enough because of a GHA bug: https://github.com/actions/checkout/issues/1471
```

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
   --consider-also-non-merged-prs    Consider also non-merged PRs (default: false)
   --minimal-delay-in-seconds value  Minimal delay in seconds between a PR and a tag (if less, we consider that the tag is always AFTER the PR) (default: 5)
   --help, -h                        show help

```