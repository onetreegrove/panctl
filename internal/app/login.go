package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	driver115 "github.com/SheltonZhu/115driver/pkg/driver"
	"github.com/justonetree/pan-cli/internal/config"
	"github.com/justonetree/pan-cli/internal/credential"
	"github.com/justonetree/pan-cli/internal/output"
	"github.com/justonetree/pan-cli/pkg/contract"
	auth115 "github.com/justonetree/pan-cli/providers/115/auth"
	client115 "github.com/justonetree/pan-cli/providers/115/client"
	model115 "github.com/justonetree/pan-cli/providers/115/model"
	clientBaidu "github.com/justonetree/pan-cli/providers/baidu/client"
	modelBaidu "github.com/justonetree/pan-cli/providers/baidu/model"
	"github.com/spf13/cobra"
)

func loginCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "login"}
	cmd.AddCommand(loginCookieCommand(rt), loginStatusCommand(rt), logoutCommand(rt), loginQRCommand(rt), loginWaitCommand(rt))
	if rt.providerName() == "baidu" {
		cmd.AddCommand(loginRefreshTokenCommand(rt))
	}
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
			providerName := rt.providerName()
			_, err := credential.NewFileStore(base).Load(providerName, rt.Profile)
			meta := contract.Meta{Provider: providerName, Profile: rt.Profile, RequestID: requestID()}
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
			providerName := rt.providerName()
			err := credential.NewFileStore(base).Delete(providerName, rt.Profile)
			meta := contract.Meta{Provider: providerName, Profile: rt.Profile, RequestID: requestID()}
			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{"logged_out": err == nil})
				os.Exit(code)
			}
			fmt.Fprintln(os.Stdout, "Logged out")
			return err
		},
	}
}

func loginQRCommand(rt *Runtime) *cobra.Command {
	var source string
	cmd := &cobra.Command{
		Use: "qr",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client115.New(2)
			sess, err := c.QRCodeStart(cmd.Context())
			if err != nil {
				handleErr(rt, contract.Meta{Provider: "115", Profile: rt.Profile, RequestID: requestID()}, err)
				return nil
			}

			sessionID := sess.UID
			base := rt.ConfigDir
			if base == "" {
				base = config.DefaultBaseDir()
			}
			dir := filepath.Join(base, "115-login")
			
			qrSess := auth115.QRSession{
				SessionID: sessionID,
				Token:     sess.UID,
				Sign:      sess.Sign,
				Time:      sess.Time,
				LoginURL:  sess.QrcodeContent,
				Source:    source,
				ExpiresAt: time.Now().Add(time.Minute * 5),
			}

			if err := auth115.SaveQRSession(dir, qrSess); err != nil {
				handleErr(rt, contract.Meta{Provider: "115", Profile: rt.Profile, RequestID: requestID()}, err)
				return nil
			}

			meta := contract.Meta{Provider: "115", Profile: rt.Profile, RequestID: requestID()}
			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"session_id": sessionID,
					"login_url":  sess.QrcodeContent,
					"expires_at": qrSess.ExpiresAt.Format(time.RFC3339),
				})
				os.Exit(code)
			}

			fmt.Fprintf(os.Stdout, "Scan this URL with your 115 App to login: %s\n", sess.QrcodeContent)
			fmt.Fprintf(os.Stdout, "After scanning, run: 115-cli login wait %s\n", sessionID)
			return nil
		},
	}
	cmd.Flags().StringVar(&source, "source", "linux", "115 login source: linux, web, android, etc.")
	return cmd
}

func loginRefreshTokenCommand(rt *Runtime) *cobra.Command {
	var refreshToken string
	var clientID string
	var clientSecret string
	var skipCheck bool
	cmd := &cobra.Command{
		Use: "refresh-token",
		RunE: func(cmd *cobra.Command, args []string) error {
			cred := modelBaidu.Credential{
				RefreshToken: refreshToken,
				ClientID:     clientID,
				ClientSecret: clientSecret,
			}.WithDefaults()
			if cred.RefreshToken == "" {
				return fmt.Errorf("missing refresh token")
			}
			if !skipCheck {
				c := clientBaidu.New(2)
				c.ImportCredential(cred)
				if err := c.RefreshToken(cmd.Context()); err != nil {
					handleErr(rt, contract.Meta{Provider: "baidu", Profile: rt.Profile, RequestID: requestID()}, err)
					return nil
				}
				cred = c.Credential()
			}
			base := rt.ConfigDir
			if base == "" {
				base = config.DefaultBaseDir()
			}
			payload, _ := json.Marshal(cred)
			if err := credential.NewFileStore(base).Save("baidu", rt.Profile, payload); err != nil {
				return err
			}
			meta := contract.Meta{Provider: "baidu", Profile: rt.Profile, RequestID: requestID()}
			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"authenticated": true,
					"refresh_token": cred.RedactedRefreshToken(),
				})
				os.Exit(code)
			}
			fmt.Fprintf(os.Stdout, "Logged in to Baidu with refresh token %s\n", cred.RedactedRefreshToken())
			return nil
		},
	}
	cmd.Flags().StringVar(&refreshToken, "token", "", "Baidu refresh token")
	cmd.Flags().StringVar(&clientID, "client-id", "", "Baidu OAuth client id")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "Baidu OAuth client secret")
	cmd.Flags().BoolVar(&skipCheck, "skip-check", false, "store credential without remote verification")
	return cmd
}

func loginWaitCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "wait <session_id>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]
			base := rt.ConfigDir
			if base == "" {
				base = config.DefaultBaseDir()
			}
			dir := filepath.Join(base, "115-login")
			qrSess, err := auth115.LoadQRSession(dir, sessionID)
			if err != nil {
				handleErr(rt, contract.Meta{Provider: "115", Profile: rt.Profile, RequestID: requestID()}, err)
				return nil
			}

			c := client115.New(2)
			sess := &driver115.QRCodeSession{
				UID:           qrSess.Token,
				Sign:          qrSess.Sign,
				Time:          qrSess.Time,
				QrcodeContent: qrSess.LoginURL,
			}

			meta := contract.Meta{Provider: "115", Profile: rt.Profile, RequestID: requestID()}
			var rawCred *driver115.Credential
			start := time.Now()
			for {
				status, err := c.QRCodeStatus(cmd.Context(), sess)
				if err != nil {
					handleErr(rt, meta, err)
					return nil
				}

				if status.IsAllowed() {
					rawCred, err = c.QRCodeLoginWithApp(cmd.Context(), sess, qrSess.Source)
					if err != nil {
						handleErr(rt, meta, err)
						return nil
					}
					break
				} else if status.IsCanceled() {
					errCanceled := contract.NewError(contract.CodeAuthExpired, "QR login was canceled by user.", "", false)
					if rt.JSON {
						code := output.WriteError(os.Stdout, os.Stderr, meta, errCanceled)
						os.Exit(code)
					}
					fmt.Fprintln(os.Stderr, "Error: Login canceled by user")
					os.Exit(contract.ExitCode(contract.CodeAuthExpired))
				} else if status.IsExpired() {
					errExpired := contract.NewError(contract.CodeAuthExpired, "QR login session expired.", "", false)
					if rt.JSON {
						code := output.WriteError(os.Stdout, os.Stderr, meta, errExpired)
						os.Exit(code)
					}
					fmt.Fprintln(os.Stderr, "Error: Login session expired")
					os.Exit(contract.ExitCode(contract.CodeAuthExpired))
				}

				if time.Since(start) > 5*time.Minute {
					errTimeout := contract.NewError(contract.CodeAuthExpired, "QR login timeout exceeded.", "", false)
					if rt.JSON {
						code := output.WriteError(os.Stdout, os.Stderr, meta, errTimeout)
						os.Exit(code)
					}
					fmt.Fprintln(os.Stderr, "Error: Timeout waiting for QR login")
					os.Exit(contract.ExitCode(contract.CodeAuthExpired))
				}

				time.Sleep(2 * time.Second)
			}

			cred := model115.Credential{
				UID:  rawCred.UID,
				CID:  rawCred.CID,
				SEID: rawCred.SEID,
				KID:  rawCred.KID,
			}

			store := credential.NewFileStore(base)
			payload, _ := json.Marshal(cred)
			if err := store.Save("115", rt.Profile, payload); err != nil {
				handleErr(rt, meta, err)
				return nil
			}

			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"authenticated": true,
					"uid":           cred.RedactedUID(),
				})
				os.Exit(code)
			}

			fmt.Fprintf(os.Stdout, "Logged in to 115 via QR as %s\n", cred.RedactedUID())
			return nil
		},
	}
	return cmd
}
