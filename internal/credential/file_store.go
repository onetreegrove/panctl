package credential

import (
	"os"
	"path/filepath"
)

type FileStore struct {
	baseDir string
}

func NewFileStore(baseDir string) *FileStore {
	return &FileStore{baseDir: baseDir}
}

func (s *FileStore) Save(providerName, profile string, data []byte) error {
	dir := filepath.Join(s.baseDir, "credentials")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(s.path(providerName, profile), data, 0600)
}

func (s *FileStore) Load(providerName, profile string) ([]byte, error) {
	return os.ReadFile(s.path(providerName, profile))
}

func (s *FileStore) Delete(providerName, profile string) error {
	err := os.Remove(s.path(providerName, profile))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *FileStore) path(providerName, profile string) string {
	if profile == "" {
		profile = "default"
	}
	return filepath.Join(s.baseDir, "credentials", providerName+"."+profile+".json")
}
