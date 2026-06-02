package app

import (
	"fmt"
	"os"
	"time"

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
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}

func requestID() string {
	return "req_" + time.Now().UTC().Format("20060102_150405")
}
