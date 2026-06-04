package cmd

import (
	"cfmux/internal/profile"
	"cfmux/internal/service"

	"github.com/spf13/cobra"
)

var serviceIsEnabledCmd = &cobra.Command{
	Use:   "is-enabled [tunnel]",
	Short: "Print whether the tunnel's service is enabled; exit code mirrors systemctl (no sudo required)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		current, err := profile.Current()
		if err != nil {
			return err
		}
		return service.IsEnabled(current, args[0])
	},
}

func init() {
	serviceCmd.AddCommand(serviceIsEnabledCmd)
}
