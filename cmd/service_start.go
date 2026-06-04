package cmd

import (
	"cfmux/internal/profile"
	"cfmux/internal/service"

	"github.com/spf13/cobra"
)

var serviceStartCmd = &cobra.Command{
	Use:   "start [tunnel]",
	Short: "Start the tunnel's systemd service (requires sudo)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		current, err := profile.Current()
		if err != nil {
			return err
		}
		return service.Control(current, args[0], "start", printProgress)
	},
}

func init() {
	serviceCmd.AddCommand(serviceStartCmd)
}
