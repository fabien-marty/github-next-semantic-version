package repocache

import (
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/fabien-marty/github-next-semantic-version/internal/app/repo"
)

const cacheVersion = 1

var _ repo.Port = &Adapter{}

type AdapterOptions struct {
	CacheLocation string
	CacheLifetime int
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

func (r *Adapter) getCacheFilePath(base string, since *time.Time, onlyMerged bool) string {
	h := sha256.New()
	key := fmt.Sprintf("%d-%s/%s-%s-%s-%t", cacheVersion, r.owner, r.repo, base, since, onlyMerged)
	h.Write([]byte(key))
	return filepath.Join(r.opts.CacheLocation, fmt.Sprintf("%x.cache", (h.Sum(nil))))
}

func (r *Adapter) GetPullRequestsSince(base string, since *time.Time, onlyMerged bool) (res []*repo.PullRequest, err error) {
	if !r.IsEnabled() {
		return r.upstreamAdapter.GetPullRequestsSince(base, since, onlyMerged)
	}
	cacheFilePath := r.getCacheFilePath(base, since, onlyMerged)
	logger := slog.Default().With(slog.String("cacheFilePath", cacheFilePath))
	info, err := os.Stat(cacheFilePath)
	if err == nil {
		if time.Since(info.ModTime()) <= time.Duration(r.opts.CacheLifetime*int(time.Second)) {
			file, err2 := os.Open(cacheFilePath)
			if err2 != nil {
				logger.Warn("can't open the cache file => cache disabled", slog.String("err", err.Error()))
			} else {
				defer file.Close()
				decoder := gob.NewDecoder(file)
				res = []*repo.PullRequest{}
				err2 = decoder.Decode(&res)
				if err2 != nil {
					logger.Warn("can't decode the cache file => cache disabled", slog.String("err", err.Error()))
				} else {
					logger.Debug("cache hit")
					return res, nil
				}
			}
		} else {
			logger.Debug("expired cache")
			err2 := os.Remove(cacheFilePath)
			if err2 != nil {
				logger.Warn("can't delete expired cache file")
			}
		}
	}
	logger.Debug("cache miss")
	res, err = r.upstreamAdapter.GetPullRequestsSince(base, since, onlyMerged)
	if err != nil {
		return res, err
	}
	file, err := os.Create(cacheFilePath)
	if err != nil {
		logger.Warn("can't create the cache file => cache disabled")
		return res, nil
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(res)
	if err != nil {
		logger.Warn("can't encode the context of the cache file => cache disabled")
		return res, nil
	}
	logger.Debug("cache saved")
	return res, nil
}

func (r *Adapter) CreateRelease(base string, tagName string, body string, draft bool) error {
	return r.upstreamAdapter.CreateRelease(base, tagName, body, draft)
}

func (r *Adapter) IsEnabled() bool {
	return r.opts.CacheLocation != ""
}
