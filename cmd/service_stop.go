package cmd

import (
	"cfmux/internal/profile"
	"cfmux/internal/service"

	"github.com/spf13/cobra"
)

var serviceStopCmd = &cobra.Command{
	Use:   "stop [tunnel]",
	Short: "Stop the tunnel's systemd service (requires sudo)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		current, err := profile.Current()
		if err != nil {
			return err
		}
		return service.Control(current, args[0], "stop", printProgress)
	},
}

func init() {
	serviceCmd.AddCommand(serviceStopCmd)
}
