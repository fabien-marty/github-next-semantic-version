package git

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTag(t *testing.T) {
	n := "1.0.0"
	tim := time.Now()
	tag := NewTag(n, &tim)
	assert.Equal(t, n, tag.Name)
	assert.Equal(t, &tim, tag.Time)
	assert.NotNil(t, tag.Semver)
	assert.Equal(t, uint64(1), tag.Semver.Major())
	assert.Equal(t, uint64(0), tag.Semver.Minor())
	assert.Equal(t, uint64(0), tag.Semver.Patch())
	assert.Equal(t, "", tag.Prefix)
}

func TestNewTagWithPrefix(t *testing.T) {
	n := "v1.0.0"
	tim := time.Now()
	tag := NewTag(n, &tim)
	assert.Equal(t, n, tag.Name)
	assert.Equal(t, &tim, tag.Time)
	assert.NotNil(t, tag.Semver)
	assert.Equal(t, uint64(1), tag.Semver.Major())
	assert.Equal(t, uint64(0), tag.Semver.Minor())
	assert.Equal(t, uint64(0), tag.Semver.Patch())
	assert.Equal(t, "v", tag.Prefix)
}

func TestNewTagWithPrefix2(t *testing.T) {
	n := "foo/bar/v1.0.0"
	tim := time.Now()
	tag := NewTag(n, &tim)
	assert.Equal(t, n, tag.Name)
	assert.Equal(t, &tim, tag.Time)
	assert.NotNil(t, tag.Semver)
	assert.Equal(t, uint64(1), tag.Semver.Major())
	assert.Equal(t, uint64(0), tag.Semver.Minor())
	assert.Equal(t, uint64(0), tag.Semver.Patch())
	assert.Equal(t, "foo/bar/v", tag.Prefix)
}

func TestNewTagWithNonSemantic(t *testing.T) {
	n := "foo"
	tim := time.Now()
	tag := NewTag(n, &tim)
	assert.Equal(t, n, tag.Name)
	assert.Equal(t, &tim, tag.Time)
	assert.Nil(t, tag.Semver)
}

func TestLessThanSameSemanticVersion(t *testing.T) {
	tim := time.Now()
	tim_1h := tim.Add(1 * time.Hour)
	t1 := NewTag("v1.0.0", &tim)
	t2 := NewTag("1.0.0", &tim_1h)
	assert.True(t, t1.LessThan(t2))
}

func TestLessThanWithNonSemantic(t *testing.T) {
	tim := time.Now()
	tim_1h := tim.Add(1 * time.Hour)
	t1 := NewTag("v1.0.0", &tim)
	t2 := NewTag("bar", &tim_1h)
	assert.False(t, t1.LessThan(t2))
}

func TestLessThan(t *testing.T) {
	tim := time.Now()
	tim_1h := tim.Add(1 * time.Hour)
	t1 := NewTag("v1.1.0", &tim)
	t2 := NewTag("v1.0.0", &tim_1h)
	assert.False(t, t1.LessThan(t2))
}

func TestNewTagIncrementingPatch(t *testing.T) {
	tim := time.Now()
	t1 := NewTag("v1.0.0", &tim)
	t2 := NewTag("1.1.0", &tim)
	nt1 := NewTagIncrementingPatch(t1)
	nt2 := NewTagIncrementingPatch(t2)
	assert.Equal(t, "v1.0.1", nt1.Name)
	assert.Equal(t, "1.1.1", nt2.Name)
}

func TestNewTagIncrementingMinor(t *testing.T) {
	tim := time.Now()
	t1 := NewTag("v1.0.1", &tim)
	t2 := NewTag("1.1.1", &tim)
	nt1 := NewTagIncrementingMinor(t1)
	nt2 := NewTagIncrementingMinor(t2)
	assert.Equal(t, "v1.1.0", nt1.Name)
	assert.Equal(t, "1.2.0", nt2.Name)
}

func TestNewTagIncrementingMajor(t *testing.T) {
	tim := time.Now()
	t1 := NewTag("v1.1.1", &tim)
	t2 := NewTag("1.1.1", &tim)
	nt1 := NewTagIncrementingMajor(t1)
	nt2 := NewTagIncrementingMajor(t2)
	assert.Equal(t, "v2.0.0", nt1.Name)
	assert.Equal(t, "2.0.0", nt2.Name)
}
