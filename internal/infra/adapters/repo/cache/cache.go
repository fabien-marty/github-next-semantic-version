package repocache

import (
	"crypto/sha256"
	"encoding/gob"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/fabien-marty/github-next-semantic-version/internal/app/repo"
)

var cacheMissErr error = errors.New("cache miss")

const cacheVersion = 1

var _ repo.Port = &Adapter{}

type AdapterOptions struct {
	CacheLocation        string
	CacheLifetime        int
	CacheDontTryToUpdate bool
}

type Adapter struct {
	owner           string
	repo            string
	upstreamAdapter repo.Port
	opts            AdapterOptions
}

func getWorkingDirectory() string {
	path, err := os.Getwd()
	if err != nil {
		panic(err.Error())
	}
	return path
}

func fixCacheLocation(cacheLocation string) string {
	logger := slog.Default().With(slog.String("cacheLocation", cacheLocation))
	if cacheLocation == "" {
		return getWorkingDirectory()
	}
	info, err := os.Stat(cacheLocation)
	if err != nil {
		logger.Warn("bad cacheLocation => cache disabled", slog.String("err", err.Error()))
		return ""
	}
	if !info.IsDir() {
		logger.Warn("bad cacheLocation, not a directory => cache disabled")
	}
	path, err := filepath.Abs(cacheLocation)
	if err != nil {
		logger.Warn("cacheLocation: can't find the absolute path => cache disabled", slog.String("err", err.Error()))
		return ""
	}
	return path
}

func fixCacheLifetime(cacheLifetime int) int {
	if cacheLifetime <= 0 {
		return 3600
	}
	return cacheLifetime
}

func NewAdapter(owner string, repo string, upstreamAdapter repo.Port, opts AdapterOptions) *Adapter {
	opts.CacheLocation = fixCacheLocation(opts.CacheLocation)
	opts.CacheLifetime = fixCacheLifetime(opts.CacheLifetime)
	return &Adapter{
		owner:           owner,
		repo:            repo,
		upstreamAdapter: upstreamAdapter,
		opts:            opts,
	}
}

func (r *Adapter) getCacheFilePath(base string, onlyMerged bool) string {
	h := sha256.New()
	key := fmt.Sprintf("%d-%s/%s-%s-%t", cacheVersion, r.owner, r.repo, base, onlyMerged)
	h.Write([]byte(key))
	return filepath.Join(r.opts.CacheLocation, fmt.Sprintf("%x.cache", (h.Sum(nil))))
}

func (r *Adapter) getPullRequestsFromCache(base string, onlyMerged bool) (res []*repo.PullRequest, err error) {
	cacheFilePath := r.getCacheFilePath(base, onlyMerged)
	logger := slog.Default().With(slog.String("cacheFilePath", cacheFilePath))
	info, err := os.Stat(cacheFilePath)
	if err != nil {
		return nil, cacheMissErr
	}
	if time.Since(info.ModTime()) > time.Duration(r.opts.CacheLifetime*int(time.Second)) {
		logger.Debug("expired cache")
		err2 := os.Remove(cacheFilePath)
		if err2 != nil {
			logger.Warn("can't delete expired cache file")
		}
		return nil, cacheMissErr
	}
	file, err2 := os.Open(cacheFilePath)
	if err2 != nil {
		logger.Warn("can't open the cache file", slog.String("err", err.Error()))
		return nil, cacheMissErr
	}
	defer file.Close()
	decoder := gob.NewDecoder(file)
	res = []*repo.PullRequest{}
	err2 = decoder.Decode(&res)
	if err2 != nil {
		logger.Warn("can't decode the cache file", slog.String("err", err.Error()))
		return nil, cacheMissErr
	}
	logger.Debug("cache hit")
	return res, nil
}

func (r *Adapter) saveCache(base string, onlyMerged bool, res []*repo.PullRequest) {
	cacheFilePath := r.getCacheFilePath(base, onlyMerged)
	logger := slog.Default().With(slog.String("cacheFilePath", cacheFilePath))
	file, err := os.Create(cacheFilePath)
	if err != nil {
		logger.Warn("can't create the cache file => cache disabled")
		return
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(res)
	if err != nil {
		logger.Warn("can't encode the context of the cache file => cache disabled")
		return
	}
	logger.Debug("cache saved")
}

func sortPRByUpdatedAt(a, b *repo.PullRequest) int {
	if (a == nil) || (b == nil) {
		panic("can't be nil")
	}
	if (a.UpdatedAt == nil) || (b.UpdatedAt == nil) {
		panic("UpdateAt can't be nil")
	}
	if a.UpdatedAt.Before(*b.UpdatedAt) {
		return -1
	} else {
		return 1
	}
}

// GetPullRequests returns pull requests from the given base branch.
// If onlyMerged is true, only merged PRs are returned.
//
// The method implements a caching strategy to avoid hitting the upstream API too often:
//
// 1. First, it tries to get PRs from the cache file
// 2. If cache miss, it gets all PRs from upstream and saves them to cache
// 3. If cache hit, it implements the following invalidation strategy:
//   - Gets the first page of most recently updated PRs from upstream
//   - If the least recently updated PR from that page matches a PR in cache
//     with the same updatedAt timestamp, the cache is considered valid
//   - Otherwise, the cache is invalidated and all PRs are fetched from upstream
//
// 4. When cache is valid, it merges:
//   - The recently updated PRs from the first page
//   - The older PRs from cache that weren't in the first page
//
// This allows keeping an up-to-date view of PRs while minimizing API calls.
//
// Note: why not using pagination on recently updated PRs?
// => because the github api is full of bugs when you combine pagination and sorting
func (r *Adapter) GetPullRequests(base string, onlyMerged bool) (res []*repo.PullRequest, err error) {
	logger := slog.Default().With(slog.String("base", base), slog.Bool("onlyMerged", onlyMerged))
	if !r.IsEnabled() {
		return r.upstreamAdapter.GetPullRequests(base, onlyMerged)
	}
	cachedPrs, err := r.getPullRequestsFromCache(base, onlyMerged)
	if err != nil {
		// cache miss
		res, err = r.upstreamAdapter.GetPullRequests(base, onlyMerged)
		if err == nil {
			r.saveCache(base, onlyMerged, res)
		}
		return res, err
	}
	// cache hit
	if r.opts.CacheDontTryToUpdate {
		return cachedPrs, nil
	}
	updatedPrs, err2 := r.GetLastUpdatedPullRequests(base, onlyMerged)
	if err2 != nil {
		return nil, err2
	}
	if len(cachedPrs) == 0 || len(updatedPrs) == 0 {
		logger.Debug("len(cachedPrs) == 0 || len(updatedPrs) == 0 => bypass cache")
		return r.upstreamAdapter.GetPullRequests(base, onlyMerged)
	}
	slices.SortFunc(updatedPrs, sortPRByUpdatedAt)
	leastUpdatedPr := updatedPrs[0]
	if leastUpdatedPr == nil || leastUpdatedPr.UpdatedAt == nil {
		panic("leastUpdatedPr is nil or has no UpdatedAt")
	}
	logger.Debug("leastUpdatedPr (in first page)", slog.Int("number", leastUpdatedPr.Number), slog.Time("updatedAt", *leastUpdatedPr.UpdatedAt))
	useCache := false
	for _, pr := range cachedPrs {
		if pr.Number == leastUpdatedPr.Number {
			if pr.UpdatedAt == nil {
				panic("UpdatedAt is nil")
			}
			if (*pr.UpdatedAt).Compare(*leastUpdatedPr.UpdatedAt) == 0 {
				logger.Debug("found a cached pr with the same number and the same updatedAt => let's continue to use the cache")
				useCache = true
			} else {
				logger.Debug("found a cached pr with the same number but a different updatedAt => bypass the cache")
			}
			break
		}
	}
	if !useCache {
		res, err = r.upstreamAdapter.GetPullRequests(base, onlyMerged)
		if err != nil {
			r.saveCache(base, onlyMerged, res)
		}
		return res, err
	}
	tmp := map[int]*repo.PullRequest{}
	for _, pr := range updatedPrs {
		tmp[pr.Number] = pr
	}
	for _, pr := range cachedPrs {
		_, ok := tmp[pr.Number]
		if !ok {
			tmp[pr.Number] = pr
		}
	}
	for _, pr := range tmp {
		res = append(res, pr)
	}
	r.saveCache(base, onlyMerged, res)
	return res, nil
}

func (r *Adapter) GetLastUpdatedPullRequests(base string, onlyMerged bool) ([]*repo.PullRequest, error) {
	// pass-through
	return r.upstreamAdapter.GetLastUpdatedPullRequests(base, onlyMerged)
}

func (r *Adapter) CreateRelease(base string, tagName string, body string, draft bool) error {
	// pass-through
	return r.upstreamAdapter.CreateRelease(base, tagName, body, draft)
}

func (r *Adapter) IsEnabled() bool {
	return r.opts.CacheLocation != ""
}
