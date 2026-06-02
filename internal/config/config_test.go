package config

import (
	"path/filepath"
	"testing"
)

func TestPathsUseProviderAndProfile(t *testing.T) {
	baseDir := t.TempDir()
	paths := NewPaths(baseDir, "115", "default")

	if got, want := paths.ProfilePath(), filepath.Join(baseDir, "profiles", "115.default.json"); got != want {
		t.Fatalf("profile path = %q, want %q", got, want)
	}
	if got, want := paths.CredentialPath(), filepath.Join(baseDir, "credentials", "115.default.json"); got != want {
		t.Fatalf("credential path = %q, want %q", got, want)
	}
}

func TestPathsDefaultEmptyProfile(t *testing.T) {
	baseDir := t.TempDir()
	paths := NewPaths(baseDir, "115", "")

	if got, want := paths.ProfilePath(), filepath.Join(baseDir, "profiles", "115.default.json"); got != want {
		t.Fatalf("profile path = %q, want %q", got, want)
	}
	if got, want := paths.CredentialPath(), filepath.Join(baseDir, "credentials", "115.default.json"); got != want {
		t.Fatalf("credential path = %q, want %q", got, want)
	}
}

func TestDefaultBaseDirUsesEnvironmentOverride(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("PAN_CLI_CONFIG_DIR", baseDir)

	if got := DefaultBaseDir(); got != baseDir {
		t.Fatalf("default base dir = %q, want %q", got, baseDir)
	}
}
