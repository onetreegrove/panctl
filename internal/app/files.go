package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/onetreegrove/panctl/internal/output"
	"github.com/onetreegrove/panctl/internal/resolver"
	"github.com/onetreegrove/panctl/pkg/contract"
	client115 "github.com/onetreegrove/panctl/providers/115/client"
	"github.com/spf13/cobra"
)

func handleErr(rt *Runtime, meta contract.Meta, err error) {
	contractErr := client115.MapError(err)
	if rt.JSON {
		code := output.WriteError(os.Stdout, os.Stderr, meta, contractErr)
		os.Exit(code)
	}
	fmt.Fprintf(os.Stderr, "Error: %s (%s)\n", contractErr.Message, contractErr.Detail)
	os.Exit(contract.ExitCode(contractErr.Code))
}

func handleOpsErr(rt *Runtime, meta contract.Meta, ops providerOps, err error) {
	contractErr := ops.MapError(err)
	if rt.JSON {
		code := output.WriteError(os.Stdout, os.Stderr, meta, contractErr)
		os.Exit(code)
	}
	fmt.Fprintf(os.Stderr, "Error: %s (%s)\n", contractErr.Message, contractErr.Detail)
	os.Exit(contract.ExitCode(contractErr.Code))
}

func filesCommand(rt *Runtime) []*cobra.Command {
	var page int
	var limit int
	var all bool
	var outputPath string
	var toDir string

	lsCmd := &cobra.Command{
		Use:   "ls [target]",
		Short: "List files and directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "/"
			if len(args) > 0 {
				target = args[0]
			}
			ops, meta, err := getProviderOps(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}
			info, err := resolver.Resolve(cmd.Context(), opsLister{ops: ops, providerName: meta.Provider}, target)
			if err != nil {
				handleOpsErr(rt, meta, ops, err)
				return nil
			}

			if info.Type == contract.FileTypeFile {
				if rt.JSON {
					code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
						"items": []contract.FileInfo{info},
					})
					os.Exit(code)
				}
				fmt.Fprintf(os.Stdout, "%s\t%d\t%s\n", info.Type, info.Size, info.Name)
				return nil
			}

			var items []contract.FileInfo
			var listRes listResult
			if all {
				pageVal := 1
				limitVal := 1000
				for {
					res, err := ops.List(cmd.Context(), info, pageVal, limitVal)
					if err != nil {
						handleOpsErr(rt, meta, ops, err)
						return nil
					}
					items = append(items, res.Items...)
					if !res.HasMore {
						break
					}
					pageVal++
				}
				listRes.Total = len(items)
			} else {
				res, err := ops.List(cmd.Context(), info, page, limit)
				if err != nil {
					handleOpsErr(rt, meta, ops, err)
					return nil
				}
				items = append(items, res.Items...)
				listRes = res
			}

			if rt.JSON {
				var code int
				if all {
					code = output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
						"items": items,
					})
				} else {
					code = output.WritePage(os.Stdout, os.Stderr, meta, map[string]any{
						"items": items,
					}, contract.Pagination{
						Page:     page,
						Limit:    limit,
						Total:    listRes.Total,
						HasMore:  listRes.HasMore,
						NextPage: listRes.NextPage,
					})
				}
				os.Exit(code)
			}

			for _, item := range items {
				fmt.Fprintf(os.Stdout, "%s\t%d\t%s\n", item.Type, item.Size, item.Name)
			}
			if !all && listRes.HasMore {
				fmt.Fprintf(os.Stdout, "(More items available, use --page %d or --all to view them)\n", listRes.NextPage)
			}
			return nil
		},
	}
	lsCmd.Flags().IntVar(&page, "page", 1, "page number")
	lsCmd.Flags().IntVar(&limit, "limit", 100, "page size")
	lsCmd.Flags().BoolVar(&all, "all", false, "list all items")

	downloadCmd := &cobra.Command{
		Use:   "download <target>",
		Short: "Download file or get download link",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			ops, meta, err := getProviderOps(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}
			info, err := resolver.Resolve(cmd.Context(), opsLister{ops: ops, providerName: meta.Provider}, target)
			if err != nil {
				handleOpsErr(rt, meta, ops, err)
				return nil
			}
			if info.Type != contract.FileTypeFile {
				handleOpsErr(rt, meta, ops, fmt.Errorf("target is not a file: %s", target))
				return nil
			}

			ua := "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.61 Safari/537.36 115Browser/27.0.5.7"
			urlStr, headers, err := ops.DownloadURL(cmd.Context(), info, ua)
			if err != nil {
				handleOpsErr(rt, meta, ops, err)
				return nil
			}

			if outputPath != "" {
				if rt.JSON {
					fmt.Fprintln(os.Stderr, "Downloading file...")
				} else {
					fmt.Printf("Downloading %s to %s...\n", info.Name, outputPath)
				}
				err = downloadFile(cmd.Context(), urlStr, headers, outputPath)
				if err != nil {
					handleOpsErr(rt, meta, ops, err)
					return nil
				}
				if rt.JSON {
					code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
						"downloaded": true,
						"path":       outputPath,
					})
					os.Exit(code)
				}
				fmt.Println("Download completed.")
				return nil
			}

			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"id":         info.ID,
					"name":       info.Name,
					"url":        urlStr,
					"headers":    headers,
					"expires_at": nil,
				})
				os.Exit(code)
			}

			fmt.Printf("URL: %s\n", urlStr)
			for k, v := range headers {
				fmt.Printf("%s: %v\n", k, v)
			}
			return nil
		},
	}
	downloadCmd.Flags().StringVar(&outputPath, "output", "", "output file path")

	mkdirCmd := &cobra.Command{
		Use:   "mkdir <parent> <name>",
		Short: "Create a directory",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			parent := args[0]
			name := args[1]
			ops, meta, err := getProviderOps(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}
			parentInfo, err := resolver.Resolve(cmd.Context(), opsLister{ops: ops, providerName: meta.Provider}, parent)
			if err != nil {
				handleOpsErr(rt, meta, ops, err)
				return nil
			}

			newDir, err := ops.Mkdir(cmd.Context(), parentInfo, name)
			if err != nil {
				handleOpsErr(rt, meta, ops, err)
				return nil
			}

			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"id":        newDir.ID,
					"name":      newDir.Name,
					"parent_id": parentInfo.ID,
				})
				os.Exit(code)
			}

			fmt.Fprintf(os.Stdout, "Created directory %s with ID %s\n", name, newDir.ID)
			return nil
		},
	}

	mvCmd := &cobra.Command{
		Use:   "mv <target...> --to <dir>",
		Short: "Move files or directories",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if toDir == "" {
				return fmt.Errorf("missing destination: specify --to <dir>")
			}
			ops, meta, err := getProviderOps(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}
			destInfo, err := resolver.Resolve(cmd.Context(), opsLister{ops: ops, providerName: meta.Provider}, toDir)
			if err != nil {
				handleOpsErr(rt, meta, ops, err)
				return nil
			}

			var files []contract.FileInfo
			var results []map[string]any
			for _, target := range args {
				info, err := resolver.Resolve(cmd.Context(), opsLister{ops: ops, providerName: meta.Provider}, target)
				if err != nil {
					handleOpsErr(rt, meta, ops, err)
					return nil
				}
				files = append(files, info)
				results = append(results, map[string]any{
					"target": target,
					"status": "ok",
				})
			}

			err = ops.Move(cmd.Context(), destInfo, files...)
			if err != nil {
				handleOpsErr(rt, meta, ops, err)
				return nil
			}

			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"results": results,
				})
				os.Exit(code)
			}

			fmt.Fprintf(os.Stdout, "Moved %d items to %s\n", len(files), toDir)
			return nil
		},
	}
	mvCmd.Flags().StringVar(&toDir, "to", "", "destination directory")

	cpCmd := &cobra.Command{
		Use:   "cp <target...> --to <dir>",
		Short: "Copy files or directories",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if toDir == "" {
				return fmt.Errorf("missing destination: specify --to <dir>")
			}
			ops, meta, err := getProviderOps(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}
			destInfo, err := resolver.Resolve(cmd.Context(), opsLister{ops: ops, providerName: meta.Provider}, toDir)
			if err != nil {
				handleOpsErr(rt, meta, ops, err)
				return nil
			}

			var files []contract.FileInfo
			var results []map[string]any
			for _, target := range args {
				info, err := resolver.Resolve(cmd.Context(), opsLister{ops: ops, providerName: meta.Provider}, target)
				if err != nil {
					handleOpsErr(rt, meta, ops, err)
					return nil
				}
				files = append(files, info)
				results = append(results, map[string]any{
					"target": target,
					"status": "ok",
				})
			}

			err = ops.Copy(cmd.Context(), destInfo, files...)
			if err != nil {
				handleOpsErr(rt, meta, ops, err)
				return nil
			}

			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"results": results,
				})
				os.Exit(code)
			}

			fmt.Fprintf(os.Stdout, "Copied %d items to %s\n", len(files), toDir)
			return nil
		},
	}
	cpCmd.Flags().StringVar(&toDir, "to", "", "destination directory")

	renameCmd := &cobra.Command{
		Use:   "rename <target> <new-name>",
		Short: "Rename a file or directory",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			newName := args[1]
			ops, meta, err := getProviderOps(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}
			info, err := resolver.Resolve(cmd.Context(), opsLister{ops: ops, providerName: meta.Provider}, target)
			if err != nil {
				handleOpsErr(rt, meta, ops, err)
				return nil
			}

			err = ops.Rename(cmd.Context(), info, newName)
			if err != nil {
				handleOpsErr(rt, meta, ops, err)
				return nil
			}

			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"id":       info.ID,
					"old_name": info.Name,
					"new_name": newName,
				})
				os.Exit(code)
			}

			fmt.Fprintf(os.Stdout, "Renamed %s to %s\n", info.Name, newName)
			return nil
		},
	}

	rmCmd := &cobra.Command{
		Use:   "rm <target...>",
		Short: "Remove files or directories",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ops, meta, err := getProviderOps(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}

			var files []contract.FileInfo
			var results []map[string]any
			for _, target := range args {
				info, err := resolver.Resolve(cmd.Context(), opsLister{ops: ops, providerName: meta.Provider}, target)
				if err != nil {
					handleOpsErr(rt, meta, ops, err)
					return nil
				}
				files = append(files, info)
				results = append(results, map[string]any{
					"target": target,
					"status": "ok",
				})
			}

			err = ops.Delete(cmd.Context(), files...)
			if err != nil {
				handleOpsErr(rt, meta, ops, err)
				return nil
			}

			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"results": results,
				})
				os.Exit(code)
			}

			fmt.Fprintf(os.Stdout, "Removed %d items\n", len(files))
			return nil
		},
	}

	uploadCmd := &cobra.Command{
		Use:   "upload <local-file-path> <target-dir>",
		Short: "Upload a local file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			localPath := args[0]
			targetDir := args[1]
			ops, meta, err := getProviderOps(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}
			destInfo, err := resolver.Resolve(cmd.Context(), opsLister{ops: ops, providerName: meta.Provider}, targetDir)
			if err != nil {
				handleOpsErr(rt, meta, ops, err)
				return nil
			}
			if destInfo.Type != contract.FileTypeDir {
				handleOpsErr(rt, meta, ops, fmt.Errorf("target is not a directory: %s", targetDir))
				return nil
			}

			progress := func(percent float64) {
				if !rt.JSON {
					fmt.Fprintf(os.Stderr, "\rUploading... %.1f%%", percent)
				}
			}
			contractFile, err := ops.Upload(cmd.Context(), localPath, destInfo, progress)
			if err != nil {
				handleOpsErr(rt, meta, ops, err)
				return nil
			}
			if !rt.JSON {
				fmt.Fprintln(os.Stderr, "\nUpload complete!")
			}

			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"file": contractFile,
				})
				os.Exit(code)
			}

			fmt.Fprintf(os.Stdout, "Uploaded File ID: %s\tSize: %d\tName: %s\n", contractFile.ID, contractFile.Size, contractFile.Name)
			return nil
		},
	}

	return []*cobra.Command{lsCmd, downloadCmd, mkdirCmd, mvCmd, cpCmd, renameCmd, rmCmd, uploadCmd}
}

func downloadFile(ctx context.Context, urlStr string, headers map[string][]string, outputPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return err
	}
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}
