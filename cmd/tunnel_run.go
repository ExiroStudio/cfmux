package cmd

import (
	"cfmux/internal/profile"
	"cfmux/internal/tunnel"

	"github.com/spf13/cobra"
)

var tunnelRunCmd = &cobra.Command{
	Use:   "run [name]",
	Short: "Run a tunnel from the current profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		current, err := profile.Current()
		if err != nil {
			return err
		}
		return tunnel.Run(current, args[0])
	},
}

func init() {
	tunnelCmd.AddCommand(tunnelRunCmd)
}
