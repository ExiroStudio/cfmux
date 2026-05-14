package cmd

import (
	"cfmux/internal/app"
	"cfmux/internal/cloudflared"
	"errors"
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
		var exitErr *app.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.Err != nil {
				fmt.Fprintln(os.Stderr, "Error:", exitErr.Err)
			}
			os.Exit(exitErr.Code)
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
