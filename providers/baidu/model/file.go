package model

import (
	"path"
	"strconv"
	"time"

	"github.com/onetreegrove/panctl/pkg/contract"
)

type File struct {
	ID        string
	Name      string
	Path      string
	IsDir     bool
	Size      int64
	MD5       string
	ThumbURL  string
	CreatedAt int64
	UpdatedAt int64
}

func (f File) ToContract(resolvedPath string) contract.FileInfo {
	if resolvedPath == "" {
		resolvedPath = f.Path
	}
	if resolvedPath == "" && f.Name != "" {
		resolvedPath = "/" + f.Name
	}
	fileType := contract.FileTypeFile
	if f.IsDir {
		fileType = contract.FileTypeDir
	}
	var createdAt *time.Time
	if f.CreatedAt > 0 {
		t := time.Unix(f.CreatedAt, 0)
		createdAt = &t
	}
	var updatedAt *time.Time
	if f.UpdatedAt > 0 {
		t := time.Unix(f.UpdatedAt, 0)
		updatedAt = &t
	}
	raw := map[string]any{}
	if f.MD5 != "" {
		raw["md5"] = f.MD5
	}
	return contract.FileInfo{
		ID:        f.ID,
		Name:      f.Name,
		Type:      fileType,
		Path:      path.Clean(resolvedPath),
		Size:      f.Size,
		ThumbURL:  f.ThumbURL,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Provider:  "baidu",
		Raw:       raw,
	}
}

type APIFile struct {
	Category       int    `json:"category"`
	FsID           int64  `json:"fs_id"`
	Size           int64  `json:"size"`
	Path           string `json:"path"`
	ServerFilename string `json:"server_filename"`
	MD5            string `json:"md5"`
	IsDir          int    `json:"isdir"`
	ServerCtime    int64  `json:"server_ctime"`
	ServerMtime    int64  `json:"server_mtime"`
	Ctime          int64  `json:"ctime"`
	Mtime          int64  `json:"mtime"`
	Thumbs         struct {
		URL3 string `json:"url3"`
	} `json:"thumbs"`
}

func FromAPIFile(src APIFile) File {
	name := src.ServerFilename
	if name == "" {
		name = path.Base(src.Path)
	}
	created := src.ServerCtime
	if created == 0 {
		created = src.Ctime
	}
	updated := src.ServerMtime
	if updated == 0 {
		updated = src.Mtime
	}
	return File{
		ID:        strconv.FormatInt(src.FsID, 10),
		Name:      name,
		Path:      src.Path,
		IsDir:     src.IsDir == 1,
		Size:      src.Size,
		MD5:       src.MD5,
		ThumbURL:  src.Thumbs.URL3,
		CreatedAt: created,
		UpdatedAt: updated,
	}
}
