package contract

import "time"

type FileType string

const (
	FileTypeFile FileType = "file"
	FileTypeDir  FileType = "dir"
)

type FileInfo struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Type      FileType       `json:"type"`
	Path      string         `json:"path,omitempty"`
	Size      int64          `json:"size"`
	SHA1      string         `json:"sha1,omitempty"`
	PickCode  string         `json:"pick_code,omitempty"`
	ThumbURL  string         `json:"thumb_url,omitempty"`
	CreatedAt *time.Time     `json:"created_at,omitempty"`
	UpdatedAt *time.Time     `json:"updated_at,omitempty"`
	Provider  string         `json:"provider"`
	Raw       map[string]any `json:"raw,omitempty"`
}
