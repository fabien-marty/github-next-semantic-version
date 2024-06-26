# github-next-semantic-version

Little CLI binary (written in golang) to guess the next semantic version only from existing git tags and recently merged pull-requests labels.  We don't use any "commit message parsing" here (only configurable PR labels).

```console
$ github-next-semantic-version --help

NAME:
   github-next-semantic-version - Compute the next semantic version with merged PRs and corresponding labels

USAGE:
   github-next-semantic-version [global options] command [command options] [localGitRepositoryPath]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --log-level value                 log level (DEBUG, INFO, WARN, ERROR) (default: "INFO") [$LOG_LEVEL]
   --log-format value                log format (text-human, text, json, json-gcp) (default: "text-human") [$LOG_FORMAT]
   --github-token value              github token [$GITHUB_TOKEN]
   --repo-owner value                repository owner (organization) [$REPO_OWNER]
   --repo-name value                 repository name (without owner/organization part) [$REPO_NAME]
   --branch value                    Branch to filter on (probably the main branch) (default: "main") [$BRANCH_NAME]
   --major-labels value              Coma separated list of PR labels to consider as major (default: "major,breaking,Type: Major") [$MAJOR_LABELS]
   --minor-labels value              Coma separated list of PR labels to consider as minor (default: "feature,Type: Feature,Type: Minor") [$MINOR_LABELS]
   --ignore-labels value             Coma separated list of PR labels to consider as ignored PRs (default: "Type: Hidden") [$MINOR_LABELS]
   --dont-increment-if-no-pr         Don't increment the version if no PR is found (or if only ignored PRs found) (default: false) [$IGNORE_LABELS]
   --consider-also-non-merged-prs    Consider also non-merged PRs (default: false)
   --minimal-delay-in-seconds value  Minimal delay in seconds between a PR and a tag (if less, we consider that the tag is always AFTER the PR) (default: 5)
   --help, -h                        show help

```