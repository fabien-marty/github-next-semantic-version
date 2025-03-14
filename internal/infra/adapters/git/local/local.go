package gitlocal

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/relvacode/iso8601"

	"github.com/fabien-marty/github-next-semantic-version/internal/app/git"
)

var _ git.Port = &Adapter{}

type AdapterOptions struct {
	LocalGitPath     string
	OriginBranchName string // default to "origin"
}

type Adapter struct {
	opts AdapterOptions
}

func NewAdapter(opts AdapterOptions) *Adapter {
	if opts.OriginBranchName == "" {
		opts.OriginBranchName = "origin"
	}
	return &Adapter{
		opts: opts,
	}
}

func lastLine(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return ""
	}
	return lines[len(lines)-1]
}

func extractGHRepoFromRemoteUrl(remoteUrl string) (owner string, repo string) {
	if strings.HasPrefix(remoteUrl, "git@github.com:") && strings.HasSuffix(remoteUrl, ".git") {
		url := strings.TrimSuffix(strings.TrimPrefix(remoteUrl, "git@github.com:"), ".git")
		tmp := strings.Split(url, "/")
		if len(tmp) != 2 {
			return "", ""
		}
		return tmp[0], tmp[1]
	}
	if strings.HasPrefix(remoteUrl, "https://") && strings.HasSuffix(remoteUrl, ".git") && strings.Contains(remoteUrl, "github.com/") {
		url := strings.TrimSuffix(strings.TrimPrefix(remoteUrl, "https://"), ".git")
		tmp := strings.Split(url, "/")
		if len(tmp) != 3 {
			return "", ""
		}
		return tmp[1], tmp[2]
	}
	return "", ""
}

func (r *Adapter) cwdOrDie() {
	if r.opts.LocalGitPath != "" && r.opts.LocalGitPath != "." {
		err := os.Chdir(r.opts.LocalGitPath)
		if err != nil {
			slog.Error("can't change the directory to %s: %v", r.opts.LocalGitPath, err)
			os.Exit(1)
		}
		slog.Debug("working directory changed", slog.String("newWorkingDirectory", r.opts.LocalGitPath))
	}
}

func (r *Adapter) executeCmdOrDie(logger *slog.Logger, cmd *exec.Cmd) string {
	logger.Debug(fmt.Sprintf("executing command: %s...", cmd.String()))
	output, err := cmd.Output()
	if err != nil {
		eerr, ok := err.(*exec.ExitError)
		if ok {
			logger.Error(fmt.Sprintf("bad exit code for command: %s", cmd.String()), slog.Int("code", eerr.ExitCode()), slog.String("stdout", string(output)), slog.String("stderr", string(eerr.Stderr)))
			os.Exit(1)
		} else {
			logger.Error(fmt.Sprintf("can't execute command: %s", cmd.String()), slog.String("err", err.Error()))
			os.Exit(2)
		}
	}
	return string(output)
}

func (r *Adapter) getTagNamesOrDie(branch string) []string {
	logger := slog.Default().With("branch", branch, "gitOperation", "getTagNames")
	args := []string{"tag"}
	if branch != "" {
		args = append(args, "--merged", "refs/remotes/"+r.opts.OriginBranchName+"/"+branch)
	}
	cmd := exec.Command("git", args...)
	output := r.executeCmdOrDie(logger, cmd)
	res := []string{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		res = append(res, strings.TrimSpace(scanner.Text()))
	}
	return res
}

func (r *Adapter) getTagDateOrDie(tagName string) time.Time {
	logger := slog.Default().With("tagName", tagName, "gitOperation", "getTagDate")
	cmd := exec.Command("git", "show", "-s", "--format=%cI", "refs/tags/"+tagName)
	output := strings.TrimSpace(r.executeCmdOrDie(logger, cmd))
	if len(strings.TrimSpace(output)) == 0 {
		logger.Error("can't get the date of the tag: empty")
		os.Exit(1)
	}
	tagDate, err := iso8601.ParseString(lastLine(output))
	if err != nil {
		logger.Error("can't parse the date of the tag", slog.String("err", err.Error()))
		os.Exit(1)
	}
	return tagDate
}

func (r *Adapter) GuessGHRepo() (owner string, repo string) {
	logger := slog.Default().With("gitOperation", "guessRepoOwner")
	r.cwdOrDie()
	cmd := exec.Command("git", "remote", "get-url", r.opts.OriginBranchName)
	output := r.executeCmdOrDie(logger, cmd)
	url := lastLine(output)
	return extractGHRepoFromRemoteUrl(url)
}

func (r *Adapter) GuessDefaultBranch() string {
	logger := slog.Default().With("gitOperation", "guessDefaultBranch")
	r.cwdOrDie()
	cmd := exec.Command("git", "remote", "show", r.opts.OriginBranchName)
	output := r.executeCmdOrDie(logger, cmd)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "HEAD branch:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmedLine, "HEAD branch:"))
		}
	}
	return ""
}

func (r *Adapter) GetContainedTags(branch string) ([]*git.Tag, error) {
	res := []*git.Tag{}
	r.cwdOrDie()
	tagNames := r.getTagNamesOrDie(branch)
	for _, tagName := range tagNames {
		tagDate := r.getTagDateOrDie(tagName)
		tag := git.NewTag(tagName, tagDate)
		res = append(res, tag)
	}
	return res, nil
}
