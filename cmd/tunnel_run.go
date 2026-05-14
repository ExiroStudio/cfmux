package cmd

import (
	"cfmux/internal/profile"
	"cfmux/internal/tunnel"

	"github.com/spf13/cobra"
)

var tunnelRunProfile string

var tunnelRunCmd = &cobra.Command{
	Use:   "run [name]",
	Short: "Run a tunnel from the current (or explicitly-named) profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prof := tunnelRunProfile
		if prof == "" {
			current, err := profile.Current()
			if err != nil {
				return err
			}
			prof = current
		}
		return tunnel.Run(prof, args[0])
	},
}

func init() {
	tunnelRunCmd.Flags().StringVar(&tunnelRunProfile, "profile", "", "profile to run the tunnel under (defaults to current)")
	tunnelCmd.AddCommand(tunnelRunCmd)
}
