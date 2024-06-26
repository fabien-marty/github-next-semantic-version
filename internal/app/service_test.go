package app

import (
	"testing"
	"time"

	"github.com/fabien-marty/github-next-semantic-version/internal/app/git"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/repo"
	"github.com/stretchr/testify/assert"
)

type gitDummyAdapter struct {
	tags []*git.Tag
}

func (d *gitDummyAdapter) GetContainedTags(branch string) ([]*git.Tag, error) {
	return d.tags, nil
}

type repoDummyAdapter struct {
	prs []repo.PullRequest
}

func (d *repoDummyAdapter) GetPullRequestsSince(base string, t time.Time, onlyMerged bool) ([]repo.PullRequest, error) {
	return d.prs, nil
}

func NewDefaultConfig() Config {
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
	version, err := service.GetNextVersion("main", true)
	assert.Nil(t, err)
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
	version, err := service.GetNextVersion("main", true)
	assert.Nil(t, err)
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
	version, err := service.GetNextVersion("main", true)
	assert.Nil(t, err)
	assert.Equal(t, "1.0.1", version)
}
