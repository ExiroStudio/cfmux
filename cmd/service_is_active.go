package cmd

import (
	"cfmux/internal/profile"
	"cfmux/internal/service"

	"github.com/spf13/cobra"
)

var serviceIsActiveCmd = &cobra.Command{
	Use:   "is-active [tunnel]",
	Short: "Print whether the tunnel's service is active; exit code mirrors systemctl (no sudo required)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		current, err := profile.Current()
		if err != nil {
			return err
		}
		return service.IsActive(current, args[0])
	},
}

func init() {
	serviceCmd.AddCommand(serviceIsActiveCmd)
}
