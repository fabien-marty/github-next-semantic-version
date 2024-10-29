package git

import (
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

// Tag represents a git tag with its name, creation time and semantic version.
type Tag struct {
	Name   string          // tag name (without modification)
	Time   time.Time       // commit time of the tag
	Semver *semver.Version // semver version read from tag name (nil if the tag name is not in the expected format)
	Prefix string          // Prefix read before the semver version
}

// NewTag creates a new Tag instance with the given name and date.
// It also parses the name to extract the semantic version of the tag:
// if the tag name has a prefix "v", the prefix will be removed before parsing the version,
// if the tag name is in the form foo/v1.2.3, the prefix part ("foo/") will be removed before parsing
// If the name is not in the expected format, the Semver field of the returned Tag will be nil.
func NewTag(name string, date time.Time) *Tag {
	if name == "" {
		return nil
	}
	prefix := ""
	nameWithoutPrefix := name
	if strings.Contains(nameWithoutPrefix, "/") {
		tmp := strings.Split(nameWithoutPrefix, "/")
		nameWithoutPrefix = tmp[len(tmp)-1]
		prefix = strings.Join(tmp[:len(tmp)-1], "/") + "/"
	}
	if nameWithoutPrefix[0] == 'v' {
		prefix += "v"
		nameWithoutPrefix = strings.TrimPrefix(nameWithoutPrefix, "v")
	}
	version, err := semver.NewVersion(nameWithoutPrefix)
	if err != nil {
		version = nil
	}
	return &Tag{
		Name:   name,
		Time:   date,
		Semver: version,
		Prefix: prefix,
	}
}

// LessThan compares the current Tag instance with another Tag instance and returns true if the current Tag is less than the other Tag.
// It compares the semantic versions of the tags and if they are equal, it compares the creation time of the tags.
func (t1 *Tag) LessThan(t2 *Tag) bool {
	if t1.Semver == nil {
		return true
	}
	if t2.Semver == nil {
		return false
	}
	if t1.Semver.Equal(t2.Semver) {
		return t1.Time.Before(t2.Time)
	}
	return t1.Semver.LessThan(t2.Semver)
}

// NewName returns the new name for the tag based on the provided new version.
// If the current tag name has a prefix ("v"...), the new name will also have the same prefix.
// Otherwise, the new name will not have any prefix.
func (t *Tag) NewName(newVersion semver.Version) string {
	return t.Prefix + newVersion.String()
}
