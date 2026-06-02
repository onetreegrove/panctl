package client

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppVersionCache(t *testing.T) {
	cachePath := filepath.Join(os.TempDir(), "pan-cli-115-appver.json")
	
	// Ensure a clean slate
	_ = os.Remove(cachePath)
	defer func() {
		_ = os.Remove(cachePath)
	}()

	c := New(0)

	// 1. Write mock unexpired cache
	wantVer := "99.9.9.9"
	cache := appVersionCache{
		Version:   wantVer,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	data, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("failed to marshal cache: %v", err)
	}
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		t.Fatalf("failed to write cache: %v", err)
	}

	// 2. Load and verify it uses cache
	gotVer := c.loadAppVersion(context.Background())
	if gotVer != wantVer {
		t.Errorf("expected cached version %q, got %q", wantVer, gotVer)
	}

	// 3. Make cache expired
	cache.ExpiresAt = time.Now().Add(-1 * time.Hour)
	data, err = json.Marshal(cache)
	if err != nil {
		t.Fatalf("failed to marshal expired cache: %v", err)
	}
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		t.Fatalf("failed to write expired cache: %v", err)
	}

	// 4. Load and verify it attempts to fetch from remote (falls back to default if remote request fails)
	gotVer2 := c.loadAppVersion(context.Background())
	if gotVer2 == "" {
		t.Error("expected fallback or fetched version, got empty string")
	}
	t.Logf("Fetched version after expiration: %s", gotVer2)
}
