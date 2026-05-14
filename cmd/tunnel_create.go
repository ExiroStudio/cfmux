package cmd

import (
	"cfmux/internal/profile"
	"cfmux/internal/tunnel"
	"fmt"

	"github.com/spf13/cobra"
)

var tunnelCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new tunnel in the current profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		current, err := profile.Current()
		if err != nil {
			return err
		}
		if err := tunnel.Create(current, args[0], printProgress); err != nil {
			return err
		}
		fmt.Println()
		fmt.Printf("Tunnel %s created in profile %s\n", args[0], current)
		fmt.Printf("Run it with: cfmux tunnel run %s\n", args[0])
		return nil
	},
}

func init() {
	tunnelCmd.AddCommand(tunnelCreateCmd)
}
