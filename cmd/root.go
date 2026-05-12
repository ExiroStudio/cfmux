package cmd

import (
	"cfmux/internal/cloudflared"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "cfmux",

	SilenceUsage:  true,
	SilenceErrors: true,

	DisableFlagParsing: true,
	TraverseChildren:   true,

	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		return cloudflared.Execute(args...)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
