package repocache

import (
	"os"
	"testing"
	"time"

	"github.com/fabien-marty/github-next-semantic-version/internal/app/repo"
	"github.com/stretchr/testify/assert"
)

type release struct {
	base    string
	tagName string
	body    string
	draft   bool
}

type repoDummyAdapter struct {
	prs                        []*repo.PullRequest
	releases                   []release
	getPullRequestsSinceCalled bool
}

func (d *repoDummyAdapter) GetPullRequestsSince(base string, onlyMerged bool) ([]*repo.PullRequest, error) {
	d.getPullRequestsSinceCalled = true
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

func TestCacheCreateRelease(t *testing.T) {
	upstreamAdapter := &repoDummyAdapter{}
	adapter := NewAdapter("owner", "repo", upstreamAdapter, AdapterOptions{})
	assert.Nil(t, adapter.CreateRelease("base", "tagName", "body", true))
	assert.Equal(t, upstreamAdapter.releases[0].base, "base")
	assert.Equal(t, upstreamAdapter.releases[0].tagName, "tagName")
	assert.Equal(t, upstreamAdapter.releases[0].body, "body")
	assert.Equal(t, upstreamAdapter.releases[0].draft, true)
}

func TestCacheLocation(t *testing.T) {
	upstreamAdapter := &repoDummyAdapter{}
	adapter := NewAdapter("owner", "repo", upstreamAdapter, AdapterOptions{CacheLocation: "foobar"})
	assert.False(t, adapter.IsEnabled())
	adapter = NewAdapter("owner", "repo", upstreamAdapter, AdapterOptions{})
	assert.True(t, adapter.IsEnabled())
	assert.NotEqual(t, adapter.opts.CacheLocation, "")
	assert.Greater(t, adapter.opts.CacheLifetime, 0)
}

func TestCacheGetPR(t *testing.T) {
	_ = os.Mkdir("./tmp", 0700)
	defer func() {
		_ = os.RemoveAll("./tmp")
	}()
	upstreamAdapter := &repoDummyAdapter{}
	adapter := NewAdapter("owner", "repo", upstreamAdapter, AdapterOptions{CacheLocation: "./tmp"})
	res, err := adapter.GetPullRequestsSince("base", false) // should cache miss
	assert.Nil(t, err)
	assert.Equal(t, len(res), 0)
	assert.True(t, upstreamAdapter.getPullRequestsSinceCalled)
	upstreamAdapter.getPullRequestsSinceCalled = false
	res, err = adapter.GetPullRequestsSince("base", false) // should cache hit
	assert.Nil(t, err)
	assert.Equal(t, len(res), 0)
	assert.False(t, upstreamAdapter.getPullRequestsSinceCalled)
	res, err = adapter.GetPullRequestsSince("base", true) // should cache miss (not the same parameters)
	assert.Nil(t, err)
	assert.Equal(t, len(res), 0)
	assert.True(t, upstreamAdapter.getPullRequestsSinceCalled)
	upstreamAdapter.getPullRequestsSinceCalled = true
}

func TestCacheGetPR2(t *testing.T) {
	_ = os.Mkdir("./tmp2", 0700)
	defer func() {
		os.RemoveAll("./tmp2")
	}()
	pr1 := &repo.PullRequest{MergedAt: &[]time.Time{time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)}[0]}
	pr2 := &repo.PullRequest{MergedAt: &[]time.Time{time.Date(2023, 2, 15, 0, 0, 0, 0, time.UTC)}[0]}
	pr2bis := &repo.PullRequest{MergedAt: &[]time.Time{time.Date(2023, 3, 1, 0, 0, 2, 0, time.UTC)}[0]} // 2 seconds after tag3
	pr3 := &repo.PullRequest{MergedAt: &[]time.Time{time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC)}[0]}
	pr4 := &repo.PullRequest{MergedAt: nil} // not merged PR
	upstreamAdapter := &repoDummyAdapter{prs: []*repo.PullRequest{pr1, pr2, pr2bis, pr3, pr4}}
	adapter := NewAdapter("owner", "repo", upstreamAdapter, AdapterOptions{CacheLocation: "./tmp2"})
	res, err := adapter.GetPullRequestsSince("base", false) // should cache miss
	assert.Nil(t, err)
	assert.Equal(t, len(res), 5)
	assert.True(t, upstreamAdapter.getPullRequestsSinceCalled)
	assert.Equal(t, res[3].MergedAt.Year(), 2023)
	upstreamAdapter.getPullRequestsSinceCalled = false
	res, err = adapter.GetPullRequestsSince("base", false) // should cache hit
	assert.Nil(t, err)
	assert.Equal(t, len(res), 5)
	assert.False(t, upstreamAdapter.getPullRequestsSinceCalled)
	assert.Equal(t, res[3].MergedAt.Year(), 2023)
}
