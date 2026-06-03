package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const DefaultAppVersion = "27.0.5.7"

type appVersionCache struct {
	Version   string    `json:"version"`
	ExpiresAt time.Time `json:"expires_at"`
}

var (
	appVersionOnce sync.Once
	cachedAppVer   string
)

// ensureAppVersion ensures the latest 115 client version is cached and set in User-Agent.
func (c *Client) ensureAppVersion(ctx context.Context) {
	appVersionOnce.Do(func() {
		cachedAppVer = c.loadAppVersion(ctx)
		c.raw.SetUserAgent(fmt.Sprintf("Mozilla/5.0 115Browser/%s", cachedAppVer))
	})
}

func (c *Client) loadAppVersion(ctx context.Context) string {
	cachePath := filepath.Join(os.TempDir(), "panctl-115-appver.json")

	// 1. Try loading version from local cache file
	if data, err := os.ReadFile(cachePath); err == nil {
		var cache appVersionCache
		if err := json.Unmarshal(data, &cache); err == nil {
			if time.Now().Before(cache.ExpiresAt) && cache.Version != "" {
				return cache.Version
			}
		}
	}

	// 2. Fetch from remote API if cache is missing or expired
	versions, err := c.raw.GetAppVersion()
	if err != nil {
		log.Printf("[115] Warning: failed to fetch app version: %v, falling back to default %s", err, DefaultAppVersion)
		return DefaultAppVersion
	}

	version := DefaultAppVersion
	for _, v := range versions {
		if v.AppName == "win" {
			version = v.Version
			break
		}
	}

	// 3. Save the new version to local cache file
	cache := appVersionCache{
		Version:   version,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if data, err := json.Marshal(cache); err == nil {
		_ = os.WriteFile(cachePath, data, 0600)
	}

	return version
}
