package git

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTag(t *testing.T) {
	n := "1.0.0"
	tim := time.Now()
	tag := NewTag(n, tim)
	assert.Equal(t, n, tag.Name)
	assert.Equal(t, tim, tag.Time)
	assert.NotNil(t, tag.Semver)
	assert.Equal(t, uint64(1), tag.Semver.Major())
	assert.Equal(t, uint64(0), tag.Semver.Minor())
	assert.Equal(t, uint64(0), tag.Semver.Patch())
}

func TestNewTagWithPrefix(t *testing.T) {
	n := "v1.0.0"
	tim := time.Now()
	tag := NewTag(n, tim)
	assert.Equal(t, n, tag.Name)
	assert.Equal(t, tim, tag.Time)
	assert.NotNil(t, tag.Semver)
	assert.Equal(t, uint64(1), tag.Semver.Major())
	assert.Equal(t, uint64(0), tag.Semver.Minor())
	assert.Equal(t, uint64(0), tag.Semver.Patch())
}

func TestNewTagWithPrefix2(t *testing.T) {
	n := "foo/bar/v1.0.0"
	tim := time.Now()
	tag := NewTag(n, tim)
	assert.Equal(t, n, tag.Name)
	assert.Equal(t, tim, tag.Time)
	assert.NotNil(t, tag.Semver)
	assert.Equal(t, uint64(1), tag.Semver.Major())
	assert.Equal(t, uint64(0), tag.Semver.Minor())
	assert.Equal(t, uint64(0), tag.Semver.Patch())
}

func TestNewTagWithNonSemantic(t *testing.T) {
	n := "foo"
	tim := time.Now()
	tag := NewTag(n, tim)
	assert.Equal(t, n, tag.Name)
	assert.Equal(t, tim, tag.Time)
	assert.Nil(t, tag.Semver)
}

func TestLessThanSameSemanticVersion(t *testing.T) {
	t1 := NewTag("v1.0.0", time.Now())
	t2 := NewTag("1.0.0", time.Now().Add(1*time.Hour))
	assert.True(t, t1.LessThan(t2))
}

func TestLessThanWithNonSemantic(t *testing.T) {
	t1 := NewTag("v1.0.0", time.Now())
	t2 := NewTag("bar", time.Now().Add(1*time.Hour))
	assert.False(t, t1.LessThan(t2))
}

func TestLessThan(t *testing.T) {
	t1 := NewTag("v1.1.0", time.Now())
	t2 := NewTag("v1.0.0", time.Now().Add(1*time.Hour))
	assert.False(t, t1.LessThan(t2))
}

func TestNewName(t *testing.T) {
	t1 := NewTag("v1.0.0", time.Now())
	t2 := NewTag("1.1.0", time.Now())
	assert.Equal(t, "v1.1.0", t1.NewName(*t2.Semver))
	assert.Equal(t, "1.0.0", t2.NewName(*t1.Semver))
}
