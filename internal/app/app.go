package app

import (
	"context"
	"fmt"
	"os"
	"time"

	client115 "github.com/justonetree/pan-cli/providers/115/client"
	commands115 "github.com/justonetree/pan-cli/providers/115/commands"
	"github.com/spf13/cobra"
)

type Options struct {
	BinaryName      string
	DefaultProvider string
}

type Runtime struct {
	Options   Options
	JSON      bool
	Profile   string
	ConfigDir string
}

func Run(opts Options) int {
	rt := &Runtime{Options: opts, Profile: "default"}
	root := &cobra.Command{
		Use:           opts.BinaryName,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().BoolVar(&rt.JSON, "json", false, "write machine-readable JSON to stdout")
	root.PersistentFlags().StringVar(&rt.Profile, "profile", "default", "profile name")
	root.PersistentFlags().StringVar(&rt.ConfigDir, "config-dir", "", "config directory")
	root.AddCommand(loginCommand(rt))
	root.AddCommand(filesCommand(rt)...)
	root.AddCommand(offlineCommand(rt))

	getClientHelper := func() (*client115.Client, error) {
		c, _, err := getClient(rt, context.Background())
		return c, err
	}
	getJSONHelper := func() bool {
		return rt.JSON
	}
	root.AddCommand(commands115.ShareCommands(getClientHelper, getJSONHelper)...)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}

func requestID() string {
	return "req_" + time.Now().UTC().Format("20060102_150405")
}

func (rt *Runtime) providerName() string {
	if rt.Options.DefaultProvider != "" {
		return rt.Options.DefaultProvider
	}
	return "115"
}
