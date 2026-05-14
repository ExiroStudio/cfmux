package cmd

import (
	"cfmux/internal/profile"
	"cfmux/internal/service"
	"fmt"

	"github.com/spf13/cobra"
)

var serviceInstallUser string

var serviceInstallCmd = &cobra.Command{
	Use:   "install [tunnel]",
	Short: "Install a systemd unit for a tunnel (requires sudo)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		current, err := profile.Current()
		if err != nil {
			return err
		}
		if err := service.Install(current, args[0], service.InstallOpts{User: serviceInstallUser}, printProgress); err != nil {
			return err
		}
		fmt.Printf("\nInstalled cfmux-%s-%s.service\n", current, args[0])
		fmt.Printf("Not yet enabled — run: cfmux service enable %s\n", args[0])
		return nil
	},
}

func init() {
	serviceInstallCmd.Flags().StringVar(&serviceInstallUser, "user", "", "user to run the service as (overrides SUDO_USER)")
	serviceCmd.AddCommand(serviceInstallCmd)
}
