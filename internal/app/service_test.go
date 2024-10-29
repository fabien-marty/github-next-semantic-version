package app

import (
	"log/slog"
	"testing"
	"time"

	"github.com/fabien-marty/github-next-semantic-version/internal/app/git"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/repo"
	"github.com/fabien-marty/slog-helpers/pkg/slogc"
	"github.com/stretchr/testify/assert"
)

type gitDummyAdapter struct {
	tags []*git.Tag
}

func (d *gitDummyAdapter) GetContainedTags(branch string) ([]*git.Tag, error) {
	return d.tags, nil
}

func (d *gitDummyAdapter) GuessGHRepo() (owner string, repo string) {
	return "foo", "bar"
}

type release struct {
	base    string
	tagName string
	body    string
	draft   bool
}

type repoDummyAdapter struct {
	prs      []repo.PullRequest
	releases []release
}

func (d *repoDummyAdapter) GetPullRequestsSince(base string, t time.Time, onlyMerged bool) ([]repo.PullRequest, error) {
	return d.prs, nil
}

func (d *repoDummyAdapter) CreateRelease(base string, tagName string, body string, draft bool) error {
	d.releases = append(d.releases, release{
		base:    base,
		tagName: tagName,
		body:    body,
		draft:   draft,
	})
	return nil
}

func NewDefaultConfig() Config {
	logger := slogc.GetLogger(
		slogc.WithLevel(slog.LevelDebug),
		slogc.WithLogFormat("text-human"),
	)
	slog.SetDefault(logger)
	return Config{
		PullRequestMajorLabels: []string{"major1", "major2"},
		PullRequestMinorLabels: []string{"minor1", "minor2"},
	}
}

func TestGetLatestSemanticTag(t *testing.T) {
	repoAdapter := &repoDummyAdapter{}
	gitAdapter := &gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("v1.0.0", time.Now()),
			git.NewTag("v1.0.1", time.Now().Add(1*time.Hour)),
		},
	}
	service := NewService(NewDefaultConfig(), repoAdapter, gitAdapter)
	tag, err := service.getLatestSemanticNonPrereleaseTag("main")
	assert.Nil(t, err)
	assert.Equal(t, "v1.0.1", tag.Name)
}

func TestGetContainedTags(t *testing.T) {
	repoAdapter := &repoDummyAdapter{}
	gitAdapter := &gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("v1.0.0", time.Now()),
			git.NewTag("v2.0.1", time.Now().Add(1*time.Hour)),
		},
	}
	config := NewDefaultConfig()
	config.TagRegex = "^v1.*"
	service := NewService(config, repoAdapter, gitAdapter)
	tags, err := service.getContainedTags("main")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(tags))
	assert.Equal(t, "v1.0.0", tags[0].Name)
}

func TestGetLatestSemanticTagWithoutSemantic(t *testing.T) {
	repoAdapter := &repoDummyAdapter{}
	gitAdapter := &gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("foo", time.Now()),
			git.NewTag("bar", time.Now().Add(1*time.Hour)),
		},
	}
	service := NewService(NewDefaultConfig(), repoAdapter, gitAdapter)
	_, err := service.getLatestSemanticNonPrereleaseTag("main")
	assert.NotNil(t, err)
}

func TestGetLatestSemanticTagWithPrerelease(t *testing.T) {
	repoAdapter := &repoDummyAdapter{}
	gitAdapter := &gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("v1.0.0", time.Now()),
			git.NewTag("v1.1.0-beta", time.Now().Add(1*time.Hour)),
		},
	}
	service := NewService(NewDefaultConfig(), repoAdapter, gitAdapter)
	tag, err := service.getLatestSemanticNonPrereleaseTag("main")
	assert.Nil(t, err)
	assert.Equal(t, "v1.0.0", tag.Name)
}

func TestGetNextVersionMinor(t *testing.T) {
	gitAdapter := &gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("v1.0.0", time.Now()),
		},
	}
	now := time.Now()
	repoAdapter := &repoDummyAdapter{
		prs: []repo.PullRequest{
			{
				Number:   1,
				Title:    "PR1",
				Labels:   []string{"foo", "bar"},
				MergedAt: &now,
			},
			{
				Number:   2,
				Title:    "PR2",
				Labels:   []string{"foo", "minor1"},
				MergedAt: &now,
			},
		},
	}
	service := NewService(NewDefaultConfig(), repoAdapter, gitAdapter)
	old, version, _, err := service.GetNextVersion("main", true)
	assert.Nil(t, err)
	assert.Equal(t, "v1.0.0", old)
	assert.Equal(t, "v1.1.0", version)
}

func TestGetNextVersionMinor2(t *testing.T) {
	gitAdapter := &gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("v1.0.0", time.Now()),
		},
	}
	now := time.Now()
	repoAdapter := &repoDummyAdapter{
		prs: []repo.PullRequest{
			{
				Number:   1,
				Title:    "PR1",
				Labels:   []string{"minor1", "bar"},
				MergedAt: &now,
			},
			{
				Number:   2,
				Title:    "PR2",
				Labels:   []string{"foo"},
				MergedAt: &now,
			},
		},
	}
	service := NewService(NewDefaultConfig(), repoAdapter, gitAdapter)
	old, version, _, err := service.GetNextVersion("main", true)
	assert.Nil(t, err)
	assert.Equal(t, "v1.0.0", old)
	assert.Equal(t, "v1.1.0", version)
}

func TestGetNextVersionMajor(t *testing.T) {
	gitAdapter := &gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("1.0.0", time.Now()),
		},
	}
	now := time.Now()
	repoAdapter := &repoDummyAdapter{
		prs: []repo.PullRequest{
			{
				Number:   1,
				Title:    "PR1",
				Labels:   []string{"foo", "minor1"},
				MergedAt: &now,
			},
			{
				Number:   2,
				Title:    "PR2",
				Labels:   []string{"foo", "major2"},
				MergedAt: &now,
			},
			{
				Number:   3,
				Title:    "PR3",
				Labels:   []string{"foo", "minor1"},
				MergedAt: &now,
			},
		},
	}
	service := NewService(NewDefaultConfig(), repoAdapter, gitAdapter)
	old, version, _, err := service.GetNextVersion("main", true)
	assert.Nil(t, err)
	assert.Equal(t, "1.0.0", old)
	assert.Equal(t, "2.0.0", version)
}

func TestGetNextVersionPatch(t *testing.T) {
	gitAdapter := &gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("1.0.0", time.Now()),
		},
	}
	repoAdapter := &repoDummyAdapter{
		prs: []repo.PullRequest{},
	}
	service := NewService(NewDefaultConfig(), repoAdapter, gitAdapter)
	old, version, _, err := service.GetNextVersion("main", true)
	assert.Nil(t, err)
	assert.Equal(t, "1.0.0", old)
	assert.Equal(t, "1.0.1", version)
}

func TestCreateRelease(t *testing.T) {
	gitAdapter := &gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("v1.0.0", time.Now()),
		},
	}
	now := time.Now()
	repoAdapter := &repoDummyAdapter{
		prs: []repo.PullRequest{
			{
				Number:   1,
				Title:    "PR1",
				Labels:   []string{"minor1", "bar"},
				MergedAt: &now,
			},
			{
				Number:   2,
				Title:    "PR2",
				Labels:   []string{"foo"},
				MergedAt: &now,
			},
		},
	}
	service := NewService(NewDefaultConfig(), repoAdapter, gitAdapter)
	err := service.CreateNextRelease("main", false, false, "- {{.Title}} (#{{.Number}})\n")
	assert.Nil(t, err)
	assert.Equal(t, 1, len(repoAdapter.releases))
	r := repoAdapter.releases[0]
	assert.Equal(t, "main", r.base)
	assert.Equal(t, "v1.1.0", r.tagName)
	assert.False(t, r.draft)
	assert.Equal(t, "- PR1 (#1)\n- PR2 (#2)\n", r.body)
}
