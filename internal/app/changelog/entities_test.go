package changelog

import (
	"testing"
	"time"

	"github.com/fabien-marty/github-next-semantic-version/internal/app/git"
	"github.com/fabien-marty/github-next-semantic-version/internal/app/repo"
	"github.com/stretchr/testify/assert"
)

func TestNewChangelog(t *testing.T) {
	t1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC)
	tag1 := git.NewTag("v1.0.0", &t1)
	tag2 := git.NewTag("v2.0.0", &t2)
	tag3 := git.NewTag("v3.0.0", &t3)

	pr1 := &repo.PullRequest{MergedAt: &[]time.Time{time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)}[0]}
	pr2 := &repo.PullRequest{MergedAt: &[]time.Time{time.Date(2023, 2, 15, 0, 0, 0, 0, time.UTC)}[0]}
	pr2bis := &repo.PullRequest{MergedAt: &[]time.Time{time.Date(2023, 3, 1, 0, 0, 2, 0, time.UTC)}[0]} // 2 seconds after tag3
	pr3 := &repo.PullRequest{MergedAt: &[]time.Time{time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC)}[0]}
	pr4 := &repo.PullRequest{MergedAt: nil} // not merged PR

	tags := []*git.Tag{tag1, tag2, tag3}
	prs := []*repo.PullRequest{pr1, pr2, pr2bis, pr3, pr4}

	changelog := New(tags, prs, Config{
		MinimalDelayInSeconds: 5,
		Future:                true,
	})

	assert.Equal(t, 4, len(changelog.Sections))

	// Check first section
	section := changelog.Sections[0]
	assert.Equal(t, tag1, section.Tag)
	assert.Equal(t, 0, len(section.Prs))

	// Check second section
	section = changelog.Sections[1]
	assert.Equal(t, tag2, section.Tag)
	assert.Equal(t, 1, len(section.Prs))
	assert.Equal(t, pr1, section.Prs[0])

	// Check third section
	section = changelog.Sections[2]
	assert.Equal(t, tag3, section.Tag)
	assert.Equal(t, 2, len(section.Prs))
	assert.Equal(t, pr2, section.Prs[0])
	assert.Equal(t, pr2bis, section.Prs[1])

	// Check fourth section
	section = changelog.Sections[3]
	assert.Nil(t, section.Tag)
	assert.Equal(t, 2, len(section.Prs))
	assert.Equal(t, pr3, section.Prs[0])
	assert.Equal(t, pr4, section.Prs[1])

	reversed := changelog.ReversedSections()

	assert.Equal(t, 4, len(reversed))

	section = reversed[0]
	assert.Nil(t, section.Tag)
	assert.Equal(t, 2, len(section.Prs))
	assert.Equal(t, pr3, section.Prs[0])
	assert.Equal(t, pr4, section.Prs[1])

	section = reversed[3]
	assert.Equal(t, tag1, section.Tag)
	assert.Equal(t, 0, len(section.Prs))

}

func TestIsPullRequestIncludedInThisSegment(t *testing.T) {
	t1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC)
	tag1 := &git.Tag{Time: &t1}
	tag2 := &git.Tag{Time: &t2}

	pr1 := &repo.PullRequest{MergedAt: &[]time.Time{time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)}[0]}
	pr2 := &repo.PullRequest{MergedAt: &[]time.Time{time.Date(2023, 2, 15, 0, 0, 0, 0, time.UTC)}[0]}
	pr3 := &repo.PullRequest{MergedAt: nil}                                                          // not merged PR
	pr4 := &repo.PullRequest{MergedAt: &[]time.Time{time.Date(2023, 2, 1, 0, 0, 2, 0, time.UTC)}[0]} // 2 seconds after tag2
	pr5 := &repo.PullRequest{MergedAt: &[]time.Time{time.Date(2023, 1, 1, 0, 0, 2, 0, time.UTC)}[0]} // 2 seconds after tag1

	tests := []struct {
		pr                    *repo.PullRequest
		tag1                  *git.Tag
		tag2                  *git.Tag
		minimalDelayInSeconds int
		expected              bool
	}{
		{pr1, tag1, tag2, 5, true},
		{pr2, tag1, tag2, 5, false},
		{pr3, tag1, tag2, 5, false},
		{pr3, tag1, nil, 5, true},
		{pr1, nil, tag2, 5, true},
		{pr2, nil, tag2, 5, false},
		{pr1, tag1, nil, 5, true},
		{pr2, tag1, nil, 5, true},
		{pr4, tag1, tag2, 0, false},
		{pr4, tag1, tag2, 5, true},
		{pr5, tag1, tag2, 0, true},
		{pr5, tag1, tag2, 5, false},
	}

	for _, test := range tests {
		result := isPullRequestIncludedInThisSegment(test.pr, test.tag1, test.tag2, test.minimalDelayInSeconds)
		assert.Equal(t, test.expected, result)
	}
}
