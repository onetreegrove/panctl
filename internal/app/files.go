package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/justonetree/pan-cli/internal/config"
	"github.com/justonetree/pan-cli/internal/credential"
	"github.com/justonetree/pan-cli/internal/output"
	"github.com/justonetree/pan-cli/internal/resolver"
	"github.com/justonetree/pan-cli/pkg/contract"
	model115 "github.com/justonetree/pan-cli/providers/115/model"
	client115 "github.com/justonetree/pan-cli/providers/115/client"
	"github.com/spf13/cobra"
)

type resolverLister struct {
	ctx context.Context
	c   *client115.Client
}

func (rl resolverLister) List(ctx context.Context, dirID string) ([]contract.FileInfo, error) {
	var allFiles []contract.FileInfo
	page := 1
	limit := 1150
	for {
		res, err := rl.c.List(ctx, dirID, page, limit)
		if err != nil {
			return nil, err
		}
		for _, item := range res.Items {
			allFiles = append(allFiles, item.ToContract(""))
		}
		if !res.HasMore {
			break
		}
		page++
	}
	return allFiles, nil
}

func getClient(rt *Runtime, ctx context.Context) (*client115.Client, contract.Meta, error) {
	base := rt.ConfigDir
	if base == "" {
		base = config.DefaultBaseDir()
	}
	store := credential.NewFileStore(base)
	data, err := store.Load("115", rt.Profile)
	meta := contract.Meta{Provider: "115", Profile: rt.Profile, RequestID: requestID()}
	if err != nil {
		return nil, meta, err
	}
	var cred model115.Credential
	if err := json.Unmarshal(data, &cred); err != nil {
		return nil, meta, err
	}
	c := client115.New(2)
	if err := c.LoginCookie(ctx, cred); err != nil {
		return nil, meta, err
	}
	return c, meta, nil
}

func handleErr(rt *Runtime, meta contract.Meta, err error) {
	contractErr := client115.MapError(err)
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
			c, meta, err := getClient(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}
			info, err := resolver.Resolve(cmd.Context(), resolverLister{ctx: cmd.Context(), c: c}, target)
			if err != nil {
				handleErr(rt, meta, err)
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
			var listRes client115.ListResult
			if all {
				pageVal := 1
				limitVal := 1150
				for {
					res, err := c.List(cmd.Context(), info.ID, pageVal, limitVal)
					if err != nil {
						handleErr(rt, meta, err)
						return nil
					}
					for _, item := range res.Items {
						items = append(items, item.ToContract(path.Join(target, item.Name)))
					}
					if !res.HasMore {
						break
					}
					pageVal++
				}
				listRes.Total = len(items)
			} else {
				res, err := c.List(cmd.Context(), info.ID, page, limit)
				if err != nil {
					handleErr(rt, meta, err)
					return nil
				}
				for _, item := range res.Items {
					items = append(items, item.ToContract(path.Join(target, item.Name)))
				}
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
			c, meta, err := getClient(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}
			info, err := resolver.Resolve(cmd.Context(), resolverLister{ctx: cmd.Context(), c: c}, target)
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}
			if info.Type != contract.FileTypeFile {
				handleErr(rt, meta, fmt.Errorf("target is not a file: %s", target))
				return nil
			}

			ua := "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.61 Safari/537.36 115Browser/27.0.5.7"
			urlStr, headers, err := c.DownloadURL(cmd.Context(), info.PickCode, ua)
			if err != nil {
				handleErr(rt, meta, err)
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
					handleErr(rt, meta, err)
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
			c, meta, err := getClient(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}
			parentInfo, err := resolver.Resolve(cmd.Context(), resolverLister{ctx: cmd.Context(), c: c}, parent)
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}

			newID, err := c.Mkdir(cmd.Context(), parentInfo.ID, name)
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}

			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"id":        newID,
					"name":      name,
					"parent_id": parentInfo.ID,
				})
				os.Exit(code)
			}

			fmt.Fprintf(os.Stdout, "Created directory %s with ID %s\n", name, newID)
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
			c, meta, err := getClient(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}
			destInfo, err := resolver.Resolve(cmd.Context(), resolverLister{ctx: cmd.Context(), c: c}, toDir)
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}

			var fileIDs []string
			var results []map[string]any
			for _, target := range args {
				info, err := resolver.Resolve(cmd.Context(), resolverLister{ctx: cmd.Context(), c: c}, target)
				if err != nil {
					handleErr(rt, meta, err)
					return nil
				}
				fileIDs = append(fileIDs, info.ID)
				results = append(results, map[string]any{
					"target": target,
					"status": "ok",
				})
			}

			err = c.Move(cmd.Context(), destInfo.ID, fileIDs...)
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}

			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"results": results,
				})
				os.Exit(code)
			}

			fmt.Fprintf(os.Stdout, "Moved %d items to %s\n", len(fileIDs), toDir)
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
			c, meta, err := getClient(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}
			destInfo, err := resolver.Resolve(cmd.Context(), resolverLister{ctx: cmd.Context(), c: c}, toDir)
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}

			var fileIDs []string
			var results []map[string]any
			for _, target := range args {
				info, err := resolver.Resolve(cmd.Context(), resolverLister{ctx: cmd.Context(), c: c}, target)
				if err != nil {
					handleErr(rt, meta, err)
					return nil
				}
				fileIDs = append(fileIDs, info.ID)
				results = append(results, map[string]any{
					"target": target,
					"status": "ok",
				})
			}

			err = c.Copy(cmd.Context(), destInfo.ID, fileIDs...)
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}

			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"results": results,
				})
				os.Exit(code)
			}

			fmt.Fprintf(os.Stdout, "Copied %d items to %s\n", len(fileIDs), toDir)
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
			c, meta, err := getClient(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}
			info, err := resolver.Resolve(cmd.Context(), resolverLister{ctx: cmd.Context(), c: c}, target)
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}

			err = c.Rename(cmd.Context(), info.ID, newName)
			if err != nil {
				handleErr(rt, meta, err)
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
			c, meta, err := getClient(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}

			var fileIDs []string
			var results []map[string]any
			for _, target := range args {
				info, err := resolver.Resolve(cmd.Context(), resolverLister{ctx: cmd.Context(), c: c}, target)
				if err != nil {
					handleErr(rt, meta, err)
					return nil
				}
				fileIDs = append(fileIDs, info.ID)
				results = append(results, map[string]any{
					"target": target,
					"status": "ok",
				})
			}

			err = c.Delete(cmd.Context(), fileIDs...)
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}

			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"results": results,
				})
				os.Exit(code)
			}

			fmt.Fprintf(os.Stdout, "Removed %d items\n", len(fileIDs))
			return nil
		},
	}

	return []*cobra.Command{lsCmd, downloadCmd, mkdirCmd, mvCmd, cpCmd, renameCmd, rmCmd}
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
