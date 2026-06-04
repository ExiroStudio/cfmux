package cmd

import (
	"cfmux/internal/profile"
	"cfmux/internal/service"

	"github.com/spf13/cobra"
)

var serviceDisableCmd = &cobra.Command{
	Use:   "disable [tunnel]",
	Short: "Disable the tunnel's systemd service, leaving the unit file in place (requires sudo)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		current, err := profile.Current()
		if err != nil {
			return err
		}
		return service.Control(current, args[0], "disable", printProgress)
	},
}

func init() {
	serviceCmd.AddCommand(serviceDisableCmd)
}
