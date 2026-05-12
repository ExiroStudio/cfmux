package cmd

import (
	"cfmux/internal/profile"
	"fmt"

	"github.com/spf13/cobra"
)

var profileUseCmd = &cobra.Command{
	Use:   "use [name]",
	Short: "Use profile",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		if err := profile.Use(args[0]); err != nil {
			return err
		}

		fmt.Println("Using profile:", args[0])

		return nil
	},
}

func init() {
	profileCmd.AddCommand(profileUseCmd)
}
