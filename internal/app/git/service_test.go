package git

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type gitDummyAdapter struct {
	tags []*Tag
}

func (d *gitDummyAdapter) GetContainedTags(branch string) ([]*Tag, error) {
	res := make([]*Tag, len(d.tags))
	copy(res, d.tags)
	return res, nil
}

func (d *gitDummyAdapter) GuessGHRepo() (owner string, repo string) {
	return "foo", "bar"
}

func (d *gitDummyAdapter) GuessDefaultBranch() string {
	return "main"
}

func TestGetContainedTags(t *testing.T) {
	now := time.Now()
	now_1h := now.Add(1 * time.Hour)
	gitAdapter := &gitDummyAdapter{
		tags: []*Tag{
			NewTag("v2.0.1", &now_1h),
			NewTag("v1.0.0", &now),
		},
	}
	s := New(gitAdapter)
	tags, err := s.GetTags([]string{"main"}, nil, "v1.*", true, true)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(tags))
	assert.Equal(t, "v1.0.0", tags[0].Name)
	tags, err = s.GetTags([]string{"main"}, nil, "", true, true)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(tags))
	assert.Equal(t, "v1.0.0", tags[0].Name)
	assert.Equal(t, "v2.0.1", tags[1].Name)
}
