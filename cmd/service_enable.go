package cmd

import (
	"cfmux/internal/profile"
	"cfmux/internal/service"

	"github.com/spf13/cobra"
)

var serviceEnableCmd = &cobra.Command{
	Use:   "enable [tunnel]",
	Short: "Enable and start the tunnel's systemd service (requires sudo)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		current, err := profile.Current()
		if err != nil {
			return err
		}
		return service.Enable(current, args[0], printProgress)
	},
}

func init() {
	serviceCmd.AddCommand(serviceEnableCmd)
}
