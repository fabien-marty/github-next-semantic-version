package gitlocal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractGHRepoFromRemoteUrl(t *testing.T) {
	owner, repo := extractGHRepoFromRemoteUrl("git@github.com:fabien-marty/github-next-semantic-version.git")
	assert.Equal(t, "fabien-marty", owner)
	assert.Equal(t, "github-next-semantic-version", repo)
	owner, repo = extractGHRepoFromRemoteUrl("git@github.com:fabien-martygithub-next-semantic-version.git")
	assert.Equal(t, "", owner)
	assert.Equal(t, "", repo)
	owner, repo = extractGHRepoFromRemoteUrl("foobar")
	assert.Equal(t, "", owner)
	assert.Equal(t, "", repo)
	owner, repo = extractGHRepoFromRemoteUrl("https://github.com/fabien-marty/github-next-semantic-version.git")
	assert.Equal(t, "fabien-marty", owner)
	assert.Equal(t, "github-next-semantic-version", repo)
	owner, repo = extractGHRepoFromRemoteUrl("https://foo@github.com/fabien-marty/github-next-semantic-version.git")
	assert.Equal(t, "fabien-marty", owner)
	assert.Equal(t, "github-next-semantic-version", repo)
}

func TestLastLine(t *testing.T) {
	assert.Equal(t, "c", lastLine("a\nb\nc"))
	assert.Equal(t, "c", lastLine("  c "))
	assert.Equal(t, "", lastLine(""))
}
