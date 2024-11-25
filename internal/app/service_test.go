package app

import (
	_ "embed"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/fabien-marty/github-next-semantic-version/internal/app/changelog"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/git"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/repo"
	"github.com/fabien-marty/slog-helpers/pkg/slogc"
	"github.com/stretchr/testify/assert"
)

type gitDummyAdapter struct {
	tags []*git.Tag
}

func (d *gitDummyAdapter) GetContainedTags(branch string) ([]*git.Tag, error) {
	res := make([]*git.Tag, len(d.tags))
	copy(res, d.tags)
	return res, nil
}

func (d *gitDummyAdapter) GuessGHRepo() (owner string, repo string) {
	return "foo", "bar"
}

func (d *gitDummyAdapter) GuessDefaultBranch() string {
	return "main"
}

type release struct {
	base    string
	tagName string
	body    string
	draft   bool
}

type repoDummyAdapter struct {
	prs      []*repo.PullRequest
	releases []release
}

func (d *repoDummyAdapter) GetPullRequestsSince(base string, t *time.Time, onlyMerged bool) ([]*repo.PullRequest, error) {
	res := make([]*repo.PullRequest, len(d.prs))
	copy(res, d.prs)
	return res, nil
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
		RepoOwner:              "foo",
		RepoName:               "bar",
		PullRequestMajorLabels: []string{"major1", "major2"},
		PullRequestMinorLabels: []string{"minor1", "minor2"},
	}
}

func TestGetLatestSemanticTag(t *testing.T) {
	now := time.Now()
	now_1h := now.Add(1 * time.Hour)
	repoService := repo.New(&repoDummyAdapter{})
	gitService := git.New(&gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("v1.0.1", &now_1h),
			git.NewTag("v1.0.0", &now),
		},
	})
	service := New(NewDefaultConfig(), repoService, gitService)
	tag, err := service.getLatestTag([]string{"main"}, true, true)
	assert.Nil(t, err)
	assert.Equal(t, "v1.0.1", tag.Name)
}

func TestGetLatestSemanticTagWithoutSemantic(t *testing.T) {
	now := time.Now()
	now_1h := now.Add(1 * time.Hour)
	repoService := repo.New(&repoDummyAdapter{})
	gitService := git.New(&gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("foo", &now),
			git.NewTag("bar", &now_1h),
		},
	})
	service := New(NewDefaultConfig(), repoService, gitService)
	_, err := service.getLatestTag([]string{"main"}, true, true)
	assert.NotNil(t, err)
}

func TestGetLatestSemanticTagWithPrerelease1(t *testing.T) {
	now := time.Now()
	now_1h := now.Add(1 * time.Hour)
	repoService := repo.New(&repoDummyAdapter{})
	gitService := git.New(&gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("v1.0.0", &now),
			git.NewTag("v1.1.0-beta", &now_1h),
		},
	})
	service := New(NewDefaultConfig(), repoService, gitService)
	tag, err := service.getLatestTag([]string{"main"}, true, true)
	assert.Nil(t, err)
	assert.Equal(t, "v1.0.0", tag.Name)
}

func TestGetLatestSemanticTagWithPrerelease2(t *testing.T) {
	now := time.Now()
	now_1h := now.Add(1 * time.Hour)
	repoService := repo.New(&repoDummyAdapter{})
	gitService := git.New(&gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("v1.0.0", &now),
			git.NewTag("v1.1.0-beta", &now_1h),
		},
	})
	service := New(NewDefaultConfig(), repoService, gitService)
	tag, err := service.getLatestTag([]string{"main"}, false, true)
	assert.Nil(t, err)
	assert.Equal(t, "v1.1.0-beta", tag.Name)
}

func TestGetNextVersionMinor(t *testing.T) {
	now := time.Now()
	gitService := git.New(&gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("v1.0.0", &now),
		},
	})
	repoService := repo.New(&repoDummyAdapter{
		prs: []*repo.PullRequest{
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
	})
	service := New(NewDefaultConfig(), repoService, gitService)
	old, version, err := service.GetNextVersion([]string{"main"}, true, false, true)
	assert.Nil(t, err)
	assert.Equal(t, "v1.0.0", old)
	assert.Equal(t, "v1.1.0", version)
}

func TestGetNextVersionMinor2(t *testing.T) {
	now := time.Now()
	gitService := git.New(&gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("v1.0.0", &now),
		},
	})
	repoService := repo.New(&repoDummyAdapter{
		prs: []*repo.PullRequest{
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
	})
	service := New(NewDefaultConfig(), repoService, gitService)
	old, version, err := service.GetNextVersion([]string{"main"}, true, false, true)
	assert.Nil(t, err)
	assert.Equal(t, "v1.0.0", old)
	assert.Equal(t, "v1.1.0", version)
}

func TestGetNextVersionMajor(t *testing.T) {
	now := time.Now()
	gitService := git.New(&gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("1.0.0", &now),
		},
	})
	repoService := repo.New(&repoDummyAdapter{
		prs: []*repo.PullRequest{
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
	})
	service := New(NewDefaultConfig(), repoService, gitService)
	old, version, err := service.GetNextVersion([]string{"main"}, true, false, true)
	assert.Nil(t, err)
	assert.Equal(t, "1.0.0", old)
	assert.Equal(t, "2.0.0", version)
}

func TestGetNextVersionMajorPrereleases(t *testing.T) {
	now := time.Now()
	gitService := git.New(&gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("1.0.0-beta", &now),
		},
	})
	repoService := repo.New(&repoDummyAdapter{
		prs: []*repo.PullRequest{
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
	})
	service := New(NewDefaultConfig(), repoService, gitService)
	old, version, err := service.GetNextVersion([]string{"main"}, true, false, false)
	assert.Nil(t, err)
	assert.Equal(t, "1.0.0-beta", old)
	assert.Equal(t, "2.0.0", version)
}

func TestGetNextVersionPatch(t *testing.T) {
	now := time.Now()
	gitService := git.New(&gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("1.0.0", &now),
		},
	})
	repoService := repo.New(&repoDummyAdapter{
		prs: []*repo.PullRequest{},
	})
	service := New(NewDefaultConfig(), repoService, gitService)
	old, version, err := service.GetNextVersion([]string{"main"}, true, false, true)
	assert.Nil(t, err)
	assert.Equal(t, "1.0.0", old)
	assert.Equal(t, "1.0.1", version)
}

func TestCreateRelease(t *testing.T) {
	now := time.Now()
	now_1h := now.Add(1 * time.Hour)
	now_2h := now.Add(2 * time.Hour)
	gitService := git.New(&gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("v1.0.0", &now_1h),
		},
	})
	repoAdapter := &repoDummyAdapter{
		prs: []*repo.PullRequest{
			{
				Number:   1,
				Title:    "PR1",
				Labels:   []string{"minor", "minor1"},
				MergedAt: &now_2h,
			},
			{
				Number:   2,
				Title:    "PR2",
				Labels:   []string{"foo"},
				MergedAt: &now_2h,
			},
		},
	}
	repoService := repo.New(repoAdapter)
	service := New(NewDefaultConfig(), repoService, gitService)
	newTag, err := service.CreateNextRelease([]string{"main"}, false, false, changelog.DefaultTemplateString, true)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(repoAdapter.releases))
	r := repoAdapter.releases[0]
	assert.Equal(t, "main", r.base)
	assert.Equal(t, "v1.1.0", r.tagName)
	assert.Equal(t, "v1.1.0", newTag)
	assert.False(t, r.draft)
	fmt.Println(r.body)
	expected := `
#### Added

- PR1 [\#1]() ([]())

#### Changed

- PR2 [\#2]() ([]())	
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(r.body))
}

func TestGenerateChangelog(t *testing.T) {
	expected := `
# CHANGELOG

## Future version **(not released)**

#### Fixed

- PR5 [\#5](https://foo.com/5) ([user9](https://foo.com/user9))

#### Changed

- PR6 [\#6](https://foo.com/6) ([user9](https://foo.com/user9))
- PR7 [\#7](https://foo.com/7) ([user8](https://foo.com/user8))


## [2.0.0](https://github.com/foo/bar/tree/2.0.0) (2024-01-02)

#### Added

- PR2 [\#2](https://foo.com/2) ([user4](https://foo.com/user4))
- PR3 [\#3](https://foo.com/3) ([user9](https://foo.com/user9))

#### Fixed

- PR4 [\#4](https://foo.com/4) ([user4](https://foo.com/user4))

<sub>[Full Diff](https://github.com/foo/bar/compare/1.0.0...2.0.0)</sub>

## [1.0.0](https://github.com/foo/bar/tree/1.0.0) (2024-01-02)

#### Fixed

- PR1 [\#1](https://foo.com/1) ([user4](https://foo.com/user4))
`
	now, _ := time.Parse("2006-01-02", "2024-01-02")
	now_1h := now.Add(1 * time.Hour)
	now_10h := now.Add(10 * time.Hour)
	now5 := now.Add(5 * time.Hour)
	now6 := now.Add(6 * time.Hour)
	now20 := now.Add(20 * time.Hour)
	gitService := git.New(&gitDummyAdapter{
		tags: []*git.Tag{
			git.NewTag("1.0.0", &now_1h),
			git.NewTag("2.0.0", &now_10h),
		},
	})
	repoService := repo.New(&repoDummyAdapter{
		prs: []*repo.PullRequest{
			{
				Number:      4,
				Title:       "PR4",
				Url:         "https://foo.com/4",
				Labels:      []string{"Type: Bug"},
				MergedAt:    &now6,
				AuthorLogin: "user4",
				AuthorUrl:   "https://foo.com/user4",
			},
			{
				Number:      1,
				Title:       "PR1",
				Url:         "https://foo.com/1",
				Labels:      []string{"foo", "Type: Bug"},
				MergedAt:    &now,
				AuthorLogin: "user4",
				AuthorUrl:   "https://foo.com/user4",
			},
			{
				Number:      2,
				Title:       "PR2",
				Url:         "https://foo.com/2",
				Labels:      []string{"Type: Major", "Type: Feature"},
				MergedAt:    &now5,
				AuthorLogin: "user4",
				AuthorUrl:   "https://foo.com/user4",
			},
			{
				Number:      3,
				Title:       "PR3",
				Url:         "https://foo.com/3",
				Labels:      []string{"foo", "Type: Feature"},
				MergedAt:    &now6,
				AuthorLogin: "user9",
				AuthorUrl:   "https://foo.com/user9",
			},
			{
				Number:      5,
				Title:       "PR5",
				Url:         "https://foo.com/5",
				Labels:      []string{"foo", "Type: Bug"},
				MergedAt:    &now20,
				AuthorLogin: "user9",
				AuthorUrl:   "https://foo.com/user9",
			},
			{
				Number:      6,
				Title:       "PR6",
				Url:         "https://foo.com/6",
				Labels:      []string{"foo", "bar"},
				MergedAt:    &now20,
				AuthorLogin: "user9",
				AuthorUrl:   "https://foo.com/user9",
			},
			{
				Number:      7,
				Title:       "PR7",
				Url:         "https://foo.com/7",
				Labels:      []string{},
				MergedAt:    &now20,
				AuthorLogin: "user8",
				AuthorUrl:   "https://foo.com/user8",
			},
		},
	})
	service := New(NewDefaultConfig(), repoService, gitService)
	res, err := service.GenerateChangelog([]string{"main"}, true, true, "", changelog.DefaultTemplateString, true)
	assert.Nil(t, err)
	fmt.Println("**********")
	fmt.Println(res)
	fmt.Println("**********")
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(res))
}
