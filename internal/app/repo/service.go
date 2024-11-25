package repo

import (
	"log/slog"
	"slices"
	"sort"
	"time"
)

type Service struct {
	adapter Port
	logger  *slog.Logger
}

func New(adapter Port) *Service {
	return &Service{
		adapter: adapter,
		logger:  slog.With("name", "repoService"),
	}
}

func (s *Service) CreateRelease(base string, tagName string, body string, draft bool) error {
	return s.adapter.CreateRelease(base, tagName, body, draft)
}

// getPullRequests returns the list of PRs merged since the given time in the given branch.
// PRs list is filtered by ignoreLabels and mustHaveLabels (if set).
// Branch can be empty and since can be nil.
func (s *Service) getPullRequests(branch string, since *time.Time, onlyMerged bool, ignoreLabels []string, mustHaveLabels []string) ([]*PullRequest, error) {
	prs, err := s.adapter.GetPullRequestsSince(branch, since, onlyMerged)
	prs = slices.DeleteFunc(prs, func(pr *PullRequest) bool {
		if pr.IsIgnored(ignoreLabels) {
			s.logger.Debug("the pr has an ignored label", slog.Int("number", pr.Number))
			return true
		}
		if len(mustHaveLabels) > 0 {
			if !pr.HasOneOfTheseLabels(mustHaveLabels) {
				s.logger.Debug("the pr doesn't have one of the required labels", slog.Int("number", pr.Number))
				return true
			}
		}
		if pr.MergedAt != nil && since != nil && pr.MergedAt.Before(*since) {
			return true
		}
		return false
	})
	return prs, err
}

// GetPullRequests returns the list of PRs merged since the given time in the given branches.
// PRs list is filtered by ignoreLabels and mustHaveLabels (if set).
// The returned slice is sorted by (ascending) mergedAt.
// Branches can be empty (no branch filtering) and since can be nil.
func (s *Service) GetPullRequests(branches []string, since *time.Time, onlyMerged bool, ignoreLabels []string, mustHaveLabels []string) ([]*PullRequest, error) {
	if len(branches) == 0 {
		return s.GetPullRequests([]string{""}, since, onlyMerged, ignoreLabels, mustHaveLabels)
	}
	res := []*PullRequest{}
	alreadyFound := map[int]bool{}
	for _, branch := range branches {
		prs, err := s.getPullRequests(branch, since, onlyMerged, ignoreLabels, mustHaveLabels)
		if err != nil {
			return nil, err
		}
		for _, pr := range prs {
			_, ok := alreadyFound[pr.Number]
			if !ok {
				alreadyFound[pr.Number] = true
				res = append(res, pr)
			}
		}
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].MergedAt == nil {
			return false
		}
		if res[j].MergedAt == nil {
			return true
		}
		return res[i].MergedAt.Before(*res[j].MergedAt)
	})
	return res, nil
}
