package config

import (
	"os"
	"path/filepath"
)

type Paths struct {
	BaseDir  string
	Provider string
	Profile  string
}

func DefaultBaseDir() string {
	if dir := os.Getenv("PANCTL_CONFIG_DIR"); dir != "" {
		return dir
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "panctl")
	}
	return filepath.Join(".", ".panctl")
}

func NewPaths(baseDir, providerName, profile string) Paths {
	if profile == "" {
		profile = "default"
	}
	return Paths{BaseDir: baseDir, Provider: providerName, Profile: profile}
}

func (p Paths) ProfilePath() string {
	return filepath.Join(p.BaseDir, "profiles", p.Provider+"."+p.Profile+".json")
}

func (p Paths) CredentialPath() string {
	return filepath.Join(p.BaseDir, "credentials", p.Provider+"."+p.Profile+".json")
}
