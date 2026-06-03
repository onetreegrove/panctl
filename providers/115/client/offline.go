package client

import (
	"context"

	driver115 "github.com/SheltonZhu/115driver/pkg/driver"
	model115 "github.com/onetreegrove/panctl/providers/115/model"
)

func (c *Client) OfflineList(ctx context.Context) ([]model115.OfflineTask, error) {
	if err := c.init(ctx); err != nil {
		return nil, err
	}
	resp, err := c.raw.ListOfflineTask(0)
	if err != nil {
		return nil, err
	}
	return convertOfflineTasks(resp.Tasks), nil
}

func (c *Client) OfflineAdd(ctx context.Context, uris []string, dstDirID string) ([]string, error) {
	if err := c.init(ctx); err != nil {
		return nil, err
	}
	// Note: We need to use raw.AddOfflineTaskURIs and pass AppVersion if required,
	// but SheltonZhu/115driver has built-in app version from options we initialized client with.
	return c.raw.AddOfflineTaskURIs(uris, dstDirID)
}

func (c *Client) OfflineDelete(ctx context.Context, hashes []string, deleteFiles bool) error {
	if err := c.init(ctx); err != nil {
		return err
	}
	return c.raw.DeleteOfflineTasks(hashes, deleteFiles)
}

func convertOfflineTasks(tasks []*driver115.OfflineTask) []model115.OfflineTask {
	var out []model115.OfflineTask
	for _, t := range tasks {
		statusText := "unknown"
		if t.IsDone() {
			statusText = "done"
		} else if t.IsFailed() {
			statusText = "failed"
		} else if t.IsRunning() {
			statusText = "running"
		} else if t.IsTodo() {
			statusText = "pending"
		}

		out = append(out, model115.OfflineTask{
			GID:        t.InfoHash,
			Name:       t.Name,
			StatusText: statusText,
			Progress:   t.Percent,
			Size:       t.Size,
			FileID:     t.FileId,
		})
	}
	return out
}
