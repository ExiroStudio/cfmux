package cmd

import (
	"cfmux/internal/profile"
	"cfmux/internal/tunnel"
	"fmt"

	"github.com/spf13/cobra"
)

var tunnelDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a tunnel from the current profile (Cloudflare + local)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		current, err := profile.Current()
		if err != nil {
			return err
		}
		if err := tunnel.Delete(current, args[0], printProgress); err != nil {
			return err
		}
		fmt.Println()
		fmt.Printf("Tunnel %s deleted from profile %s\n", args[0], current)
		return nil
	},
}

func init() {
	tunnelCmd.AddCommand(tunnelDeleteCmd)
}
