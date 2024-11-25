package repo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type repoDummyAdapter struct {
	prs []*PullRequest
}

func (d *repoDummyAdapter) GetPullRequestsSince(base string, t *time.Time, onlyMerged bool) ([]*PullRequest, error) {
	res := make([]*PullRequest, len(d.prs))
	copy(res, d.prs)
	return res, nil
}

func (d *repoDummyAdapter) CreateRelease(base string, tagName string, body string, draft bool) error {
	return nil
}

func TestGetPullRequests(t *testing.T) {
	now := time.Now()
	now_plus_1h := now.Add(time.Duration(time.Hour))
	now_plus_2h := now.Add(time.Duration(2 * time.Hour))
	s := New(&repoDummyAdapter{
		prs: []*PullRequest{
			{
				Number:   1,
				Title:    "PR1",
				Labels:   []string{"foo", "minor1"},
				MergedAt: &now_plus_1h,
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
				MergedAt: &now_plus_2h,
			},
		},
	})
	prs, err := s.GetPullRequests(nil, nil, true, nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(prs))
	assert.Equal(t, 2, prs[0].Number)
	assert.Equal(t, 1, prs[1].Number)
	assert.Equal(t, 3, prs[2].Number)
	prs, err = s.GetPullRequests([]string{"foo", "bar"}, nil, true, []string{"dummy", "major2"}, []string{"foo", "bar"})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(prs))
	assert.Equal(t, 1, prs[0].Number)
	assert.Equal(t, 3, prs[1].Number)
	prs, err = s.GetPullRequests([]string{"foo", "bar"}, nil, true, nil, []string{"bar"})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(prs))
}
