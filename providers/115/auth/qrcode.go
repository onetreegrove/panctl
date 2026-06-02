package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type QRSession struct {
	SessionID string    `json:"session_id"`
	Token     string    `json:"token"`
	Sign      string    `json:"sign"`
	Time      int64     `json:"time"`
	LoginURL  string    `json:"login_url"`
	Source    string    `json:"source"`
	ExpiresAt time.Time `json:"expires_at"`
}

func SaveQRSession(dir string, s QRSession) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, s.SessionID+".json"), b, 0600)
}

func LoadQRSession(dir, sessionID string) (QRSession, error) {
	b, err := os.ReadFile(filepath.Join(dir, sessionID+".json"))
	if err != nil {
		return QRSession{}, err
	}
	var s QRSession
	if err := json.Unmarshal(b, &s); err != nil {
		return QRSession{}, err
	}
	return s, nil
}
