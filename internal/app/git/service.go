package git

import (
	"fmt"
	"log/slog"
	"regexp"
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
		logger:  slog.With("name", "gitService"),
	}
}

func (s *Service) GuessGHRepo() (owner string, repo string) {
	return s.adapter.GuessGHRepo()
}

func (s *Service) GuessDefaultBranch() string {
	return s.adapter.GuessDefaultBranch()
}

// getTags returns the list of tags "contained" by the given branch defined after the given time
// (the list from the adapter is optionally filtered by the tag-regex value if not nil)
// (the branch can be empty, since can be nil)
func (s *Service) getTags(branch string, since *time.Time, tagRegex string, ignorePrereleases bool, ignoreNonSemantic bool) ([]*Tag, error) {
	res, err := s.adapter.GetContainedTags(branch)
	if err != nil {
		return nil, err
	}
	regex, err := regexp.Compile(tagRegex)
	if err != nil {
		return res, fmt.Errorf("can't compile the regex %s: %w", tagRegex, err)
	}
	res = slices.DeleteFunc(res, func(tag *Tag) bool {
		if !regex.MatchString(tag.Name) {
			s.logger.Debug("tag doesn't match the regex => ignoring", slog.String("name", tag.Name), slog.String("regex", tagRegex))
			return true
		}
		if ignoreNonSemantic && tag.Semver == nil {
			s.logger.Debug("tag doesn't have a semantic version => ignoring", slog.String("name", tag.Name))
			return true
		}
		if ignorePrereleases && tag.Semver.Prerelease() != "" {
			s.logger.Debug("tag is a prelease => ignoring", slog.String("name", tag.Name))
			return true
		}
		if since != nil && tag.Time != nil && (tag.Time.Before(*since) || tag.Time.Equal(*since)) {
			s.logger.Debug("tag too old => ignoring", slog.String("name", tag.Name), slog.String("time", tag.Time.Format(time.RFC3339)))
			return true
		}
		return false
	})
	return res, nil
}

// GetTags returns the list (without duplicates) of tags "contained" by the given branches defined after the given time
// (the list from the adapter is optionally filtered by the tag-regex value if not nil)
// (branches can be empty (no branch filtering), since can be nil)
// (the returned slice is sorted by (ascending) semantic version)
func (s *Service) GetTags(branches []string, since *time.Time, tagRegex string, ignorePrereleases bool, ignoreNonSemantic bool) ([]*Tag, error) {
	if len(branches) == 0 {
		return s.GetTags([]string{""}, since, tagRegex, ignorePrereleases, ignoreNonSemantic)
	}
	res := []*Tag{}
	alreadyFound := map[string]bool{}
	for _, branch := range branches {
		tags, err := s.getTags(branch, since, tagRegex, ignorePrereleases, ignoreNonSemantic)
		if err != nil {
			return nil, err
		}
		for _, tag := range tags {
			_, ok := alreadyFound[tag.Name]
			if !ok {
				alreadyFound[tag.Name] = true
				res = append(res, tag)
			}
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].LessThan(res[j])
	})
	return res, nil
}
