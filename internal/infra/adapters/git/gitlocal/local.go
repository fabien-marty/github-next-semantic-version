package gitlocal

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/relvacode/iso8601"

	"github.com/fabien-marty/github-next-semantic-version/internal/app/git"
)

const (
	prefix = "@@@PREFIX@@@"
	suffix = "@@@SUFFIX@@@"
	sep    = ""
	tag    = "~~~"
)

var _ git.Port = &Adapter{}

type AdapterOptions struct {
	LocalGitPath string
}

type Adapter struct {
	opts AdapterOptions
}

func NewAdapter(opts AdapterOptions) *Adapter {
	return &Adapter{
		opts: opts,
	}
}

func (r *Adapter) decode(output string) ([]*git.Tag, error) {
	res := []*git.Tag{}
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, prefix+tag) {
			continue
		}
		if !strings.Contains(line, suffix) {
			continue
		}
		tmp := strings.Split(line, suffix)
		if len(tmp) != 2 {
			continue
		}
		tagDate, err := iso8601.ParseString(tmp[1])
		if err != nil {
			slog.Debug("bad iso8601 date parsing => ignoring", slog.String("date", tmp[1]), slog.String("err", err.Error()))
			continue
		}
		tmp = strings.Split(tmp[0], tag)
		for i := 1; i < len(tmp); i++ {
			tagName := tmp[i]
			res = append(res, git.NewTag(tagName, tagDate))
		}
	}
	return res, nil
}

func (r *Adapter) GetContainedTags(branch string) ([]*git.Tag, error) {
	if r.opts.LocalGitPath != "" && r.opts.LocalGitPath != "." {
		err := os.Chdir(r.opts.LocalGitPath)
		if err != nil {
			return nil, fmt.Errorf("can't change the directory to %s: %v", r.opts.LocalGitPath, err)
		}
	}
	format := fmt.Sprintf("%s(decorate:prefix=%s,suffix=%s,tag=%s,separator=)%s", "%", prefix, suffix, tag, "%cI")
	cmd := exec.Command("git", "log", "--tags", "--simplify-by-decoration", fmt.Sprintf(`--pretty=%s`, format), branch)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	tags, err := r.decode(string(output))
	if err != nil {
		return nil, fmt.Errorf("can't get the list of tags from git: %v", err)
	}
	return tags, nil
}
