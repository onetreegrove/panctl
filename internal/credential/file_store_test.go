package credential

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreRoundTripAndPermission(t *testing.T) {
	store := NewFileStore(t.TempDir())
	secret := []byte(`{"uid":"u","cid":"c","seid":"s","kid":"k"}`)

	if err := store.Save("115", "default", secret); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := store.Load("115", "default")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if string(got) != string(secret) {
		t.Fatalf("secret = %s, want %s", got, secret)
	}

	stat, err := os.Stat(filepath.Join(store.baseDir, "credentials", "115.default.json"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if stat.Mode().Perm() != 0600 {
		t.Fatalf("mode = %v, want 0600", stat.Mode().Perm())
	}
}

func TestFileStoreDefaultsEmptyProfile(t *testing.T) {
	store := NewFileStore(t.TempDir())
	secret := []byte(`{"uid":"u"}`)

	if err := store.Save("115", "", secret); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := store.Load("115", "default")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if string(got) != string(secret) {
		t.Fatalf("secret = %s, want %s", got, secret)
	}
}

func TestFileStoreDeleteRemovesCredentialAndIgnoresMissing(t *testing.T) {
	store := NewFileStore(t.TempDir())

	if err := store.Save("115", "default", []byte(`{"uid":"u"}`)); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := store.Delete("115", "default"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.Load("115", "default"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("load deleted credential error = %v, want os.ErrNotExist", err)
	}
	if err := store.Delete("115", "default"); err != nil {
		t.Fatalf("delete missing: %v", err)
	}
}
