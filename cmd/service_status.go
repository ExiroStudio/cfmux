package cmd

import (
	"cfmux/internal/profile"
	"cfmux/internal/service"

	"github.com/spf13/cobra"
)

var serviceStatusCmd = &cobra.Command{
	Use:   "status [tunnel]",
	Short: "Show systemctl status for the tunnel's service (no sudo required)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		current, err := profile.Current()
		if err != nil {
			return err
		}
		return service.Status(current, args[0])
	},
}

func init() {
	serviceCmd.AddCommand(serviceStatusCmd)
}
