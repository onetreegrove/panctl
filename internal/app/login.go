package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/justonetree/pan-cli/internal/config"
	"github.com/justonetree/pan-cli/internal/credential"
	"github.com/justonetree/pan-cli/internal/output"
	"github.com/justonetree/pan-cli/pkg/contract"
	auth115 "github.com/justonetree/pan-cli/providers/115/auth"
	client115 "github.com/justonetree/pan-cli/providers/115/client"
	"github.com/spf13/cobra"
)

func loginCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "login"}
	cmd.AddCommand(loginCookieCommand(rt), loginStatusCommand(rt), logoutCommand(rt))
	return cmd
}

func loginCookieCommand(rt *Runtime) *cobra.Command {
	var fromStdin bool
	var rawCookie string
	cmd := &cobra.Command{
		Use: "cookie",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromStdin {
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}
				rawCookie = string(b)
			}
			cred, err := auth115.ParseCookie(rawCookie)
			if err != nil {
				return err
			}
			c := client115.New(2)
			if err := c.LoginCookie(cmd.Context(), cred); err != nil {
				return err
			}
			base := rt.ConfigDir
			if base == "" {
				base = config.DefaultBaseDir()
			}
			store := credential.NewFileStore(base)
			payload, _ := json.Marshal(cred)
			if err := store.Save("115", rt.Profile, payload); err != nil {
				return err
			}
			meta := contract.Meta{Provider: "115", Profile: rt.Profile, RequestID: requestID()}
			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"authenticated": true,
					"uid":           cred.RedactedUID(),
				})
				os.Exit(code)
			}
			fmt.Fprintf(os.Stdout, "Logged in to 115 as %s\n", cred.RedactedUID())
			return nil
		},
	}
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "read cookie from stdin")
	cmd.Flags().StringVar(&rawCookie, "cookie", "", "115 cookie")
	return cmd
}

func loginStatusCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use: "status",
		RunE: func(cmd *cobra.Command, args []string) error {
			base := rt.ConfigDir
			if base == "" {
				base = config.DefaultBaseDir()
			}
			_, err := credential.NewFileStore(base).Load("115", rt.Profile)
			meta := contract.Meta{Provider: "115", Profile: rt.Profile, RequestID: requestID()}
			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{"authenticated": err == nil})
				os.Exit(code)
			}
			if err != nil {
				fmt.Fprintln(os.Stdout, "Not logged in")
			} else {
				fmt.Fprintln(os.Stdout, "Logged in")
			}
			return nil
		},
	}
}

func logoutCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use: "logout",
		RunE: func(cmd *cobra.Command, args []string) error {
			base := rt.ConfigDir
			if base == "" {
				base = config.DefaultBaseDir()
			}
			err := credential.NewFileStore(base).Delete("115", rt.Profile)
			meta := contract.Meta{Provider: "115", Profile: rt.Profile, RequestID: requestID()}
			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{"logged_out": err == nil})
				os.Exit(code)
			}
			fmt.Fprintln(os.Stdout, "Logged out")
			return err
		},
	}
}
