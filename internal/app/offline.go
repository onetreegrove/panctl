package app

import (
	"fmt"
	"os"
	"time"

	"github.com/justonetree/pan-cli/internal/output"
	"github.com/justonetree/pan-cli/internal/resolver"
	"github.com/justonetree/pan-cli/pkg/contract"
	"github.com/spf13/cobra"
)

func offlineCommand(rt *Runtime) *cobra.Command {
	var toDir string
	var deleteFiles bool
	var timeout time.Duration

	cmd := &cobra.Command{Use: "offline"}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List offline tasks",
		RunE: runOfflineList(rt),
	}

	addCmd := &cobra.Command{
		Use:   "add <url>",
		Short: "Add offline download task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if rt.providerName() != "115" {
				handleErr(rt, contract.Meta{Provider: rt.providerName(), Profile: rt.Profile, RequestID: requestID()}, fmt.Errorf("offline task is not supported by %s", rt.providerName()))
				return nil
			}
			url := args[0]
			c, meta, err := getClient(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}

			dstID := "0"
			if toDir != "" {
				info, err := resolver.Resolve(cmd.Context(), resolverLister{ctx: cmd.Context(), c: c}, toDir)
				if err != nil {
					handleErr(rt, meta, err)
					return nil
				}
				dstID = info.ID
			}

			hashes, err := c.OfflineAdd(cmd.Context(), []string{url}, dstID)
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}

			gid := ""
			if len(hashes) > 0 {
				gid = hashes[0]
			}

			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"gid": gid,
				})
				os.Exit(code)
			}

			fmt.Fprintf(os.Stdout, "Added offline task, GID: %s\n", gid)
			return nil
		},
	}
	addCmd.Flags().StringVar(&toDir, "to", "", "destination directory")

	deleteCmd := &cobra.Command{
		Use:   "delete <gid>",
		Short: "Delete offline download task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if rt.providerName() != "115" {
				handleErr(rt, contract.Meta{Provider: rt.providerName(), Profile: rt.Profile, RequestID: requestID()}, fmt.Errorf("offline task is not supported by %s", rt.providerName()))
				return nil
			}
			gid := args[0]
			c, meta, err := getClient(rt, cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}

			err = c.OfflineDelete(cmd.Context(), []string{gid}, deleteFiles)
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}

			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"deleted": true,
				})
				os.Exit(code)
			}

			fmt.Fprintf(os.Stdout, "Deleted offline task: %s\n", gid)
			return nil
		},
	}
	deleteCmd.Flags().BoolVar(&deleteFiles, "delete-files", false, "delete downloaded files from disk")

	waitCmd := &cobra.Command{
		Use:   "wait <gid>",
		Short: "Wait for an offline task to complete",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gid := args[0]
			return runOfflineWait(rt, gid, 5*time.Second, timeout)(cmd, args)
		},
	}
	waitCmd.Flags().DurationVar(&timeout, "timeout", 2*time.Hour, "timeout duration")

	cmd.AddCommand(listCmd, addCmd, deleteCmd, waitCmd)
	return cmd
}

func runOfflineList(rt *Runtime) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if rt.providerName() != "115" {
			handleErr(rt, contract.Meta{Provider: rt.providerName(), Profile: rt.Profile, RequestID: requestID()}, fmt.Errorf("offline task is not supported by %s", rt.providerName()))
			return nil
		}
		c, meta, err := getClient(rt, cmd.Context())
		if err != nil {
			handleErr(rt, meta, err)
			return nil
		}

		tasks, err := c.OfflineList(cmd.Context())
		if err != nil {
			handleErr(rt, meta, err)
			return nil
		}

		var contractTasks []contract.OfflineTask
		for _, t := range tasks {
			contractTasks = append(contractTasks, t.ToContract())
		}

		if rt.JSON {
			code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
				"tasks": contractTasks,
			})
			os.Exit(code)
		}

		for _, t := range contractTasks {
			fmt.Fprintf(os.Stdout, "%s\t%s\t%s\t%.1f%%\n", t.GID, t.Status, t.Name, t.Progress)
		}
		return nil
	}
}

func runOfflineWait(rt *Runtime, gid string, interval, timeout time.Duration) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if rt.providerName() != "115" {
			handleErr(rt, contract.Meta{Provider: rt.providerName(), Profile: rt.Profile, RequestID: requestID()}, fmt.Errorf("offline task is not supported by %s", rt.providerName()))
			return nil
		}
		c, meta, err := getClient(rt, cmd.Context())
		if err != nil {
			handleErr(rt, meta, err)
			return nil
		}

		start := time.Now()
		for {
			tasks, err := c.OfflineList(cmd.Context())
			if err != nil {
				handleErr(rt, meta, err)
				return nil
			}

			var foundTask *contract.OfflineTask
			for _, t := range tasks {
				ct := t.ToContract()
				if ct.GID == gid {
					foundTask = &ct
					break
				}
			}

			if foundTask == nil {
				errNotFound := contract.NewError(contract.CodeNotFound, "Offline task not found.", "GID: "+gid, false)
				if rt.JSON {
					code := output.WriteError(os.Stdout, os.Stderr, meta, errNotFound)
					os.Exit(code)
				}
				fmt.Fprintf(os.Stderr, "Error: Task GID %s not found\n", gid)
				os.Exit(contract.ExitCode(contract.CodeNotFound))
			}

			if foundTask.Status == contract.OfflineDone {
				if rt.JSON {
					code := output.WriteOK(os.Stdout, os.Stderr, meta, foundTask)
					os.Exit(code)
				}
				fmt.Fprintf(os.Stdout, "Task completed: %s\n", foundTask.Name)
				return nil
			}

			if foundTask.Status == contract.OfflineFailed {
				errFailed := contract.NewError(contract.CodeRemoteError, "Offline task failed.", foundTask.Name, false)
				if rt.JSON {
					code := output.WriteError(os.Stdout, os.Stderr, meta, errFailed)
					os.Exit(code)
				}
				fmt.Fprintf(os.Stderr, "Error: Task %s failed\n", gid)
				os.Exit(contract.ExitCode(contract.CodeRemoteError))
			}

			if time.Since(start) > timeout {
				errTimeout := contract.NewError(contract.CodeNetworkError, "Offline task wait timeout.", fmt.Sprintf("Timeout %s exceeded", timeout), false)
				if rt.JSON {
					code := output.WriteError(os.Stdout, os.Stderr, meta, errTimeout)
					os.Exit(code)
				}
				fmt.Fprintf(os.Stderr, "Error: Timeout waiting for task %s\n", gid)
				os.Exit(contract.ExitCode(contract.CodeNetworkError))
			}

			select {
			case <-cmd.Context().Done():
				return cmd.Context().Err()
			case <-time.After(interval):
			}
		}
	}
}
