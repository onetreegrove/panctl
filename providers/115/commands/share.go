package commands

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/justonetree/pan-cli/internal/output"
	"github.com/justonetree/pan-cli/pkg/contract"
	client115 "github.com/justonetree/pan-cli/providers/115/client"
	"github.com/spf13/cobra"
)

func ShareCommands(getClient func() (*client115.Client, error), getJSON func() bool) []*cobra.Command {
	var page int
	var limit int

	lsCmd := &cobra.Command{
		Use:   "share-ls <share-url> [dir-id]",
		Short: "List files inside a shared link",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient()
			if err != nil {
				return err
			}
			shareURL := args[0]
			var dirID string
			if len(args) > 1 {
				dirID = args[1]
			}
			shareCode, receiveCode, err := client115.ParseShareURL(shareURL)
			if err != nil {
				return err
			}

			files, total, err := c.ShareList(cmd.Context(), shareCode, receiveCode, dirID, page, limit)
			if err != nil {
				return err
			}

			if getJSON() {
				code := output.WriteOK(os.Stdout, os.Stderr, contract.Meta{Provider: "115", Profile: "default"}, map[string]any{
					"items": files,
					"total": total,
				})
				os.Exit(code)
			}

			for _, f := range files {
				typeStr := "F"
				if f.Type == contract.FileTypeDir {
					typeStr = "D"
				}
				fmt.Printf("%s\t%s\t%d\t%s\n", typeStr, f.ID, f.Size, f.Name)
			}
			return nil
		},
	}
	lsCmd.Flags().IntVar(&page, "page", 1, "page number")
	lsCmd.Flags().IntVar(&limit, "limit", 100, "page size")

	var outputDest string
	downloadCmd := &cobra.Command{
		Use:   "share-download <share-url> <file-id>",
		Short: "Download file from a shared link",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient()
			if err != nil {
				return err
			}
			shareURL := args[0]
			fileID := args[1]

			shareCode, receiveCode, err := client115.ParseShareURL(shareURL)
			if err != nil {
				return err
			}

			downloadURL, err := c.ShareDownloadURL(cmd.Context(), shareCode, receiveCode, fileID)
			if err != nil {
				return err
			}

			if getJSON() {
				code := output.WriteOK(os.Stdout, os.Stderr, contract.Meta{Provider: "115", Profile: "default"}, map[string]any{
					"url": downloadURL,
					"headers": map[string]string{
						"User-Agent": "Mozilla/5.0",
					},
				})
				os.Exit(code)
			}

			if outputDest == "" {
				fmt.Println(downloadURL)
				return nil
			}

			// Download shared file
			req, err := http.NewRequestWithContext(cmd.Context(), "GET", downloadURL, nil)
			if err != nil {
				return err
			}
			req.Header.Set("User-Agent", "Mozilla/5.0")
			req.Header.Set("Referer", fmt.Sprintf("https://115cdn.com/s/%s?password=%s&", shareCode, receiveCode))

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("download failed: http status %s", resp.Status)
			}

			if err := os.MkdirAll(filepath.Dir(outputDest), 0755); err != nil {
				return err
			}
			outF, err := os.Create(outputDest)
			if err != nil {
				return err
			}
			defer outF.Close()

			_, err = io.Copy(outF, resp.Body)
			return err
		},
	}
	downloadCmd.Flags().StringVarP(&outputDest, "output", "o", "", "local file destination path")

	return []*cobra.Command{lsCmd, downloadCmd}
}
