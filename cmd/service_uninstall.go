package cmd

import (
	"cfmux/internal/profile"
	"cfmux/internal/service"
	"fmt"

	"github.com/spf13/cobra"
)

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall [tunnel]",
	Short: "Remove the systemd unit for a tunnel (requires sudo)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		current, err := profile.Current()
		if err != nil {
			return err
		}
		if err := service.Uninstall(current, args[0], printProgress); err != nil {
			return err
		}
		fmt.Printf("\nUninstalled cfmux-%s-%s.service (tunnel registry untouched)\n", current, args[0])
		return nil
	},
}

func init() {
	serviceCmd.AddCommand(serviceUninstallCmd)
}
