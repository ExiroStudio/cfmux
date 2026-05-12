package cmd

import (
	"cfmux/internal/profile"
	"fmt"

	"github.com/spf13/cobra"
)

var profileCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current profile",

	RunE: func(cmd *cobra.Command, args []string) error {
		current, err := profile.Current()
		if err != nil {
			return err
		}

		fmt.Println(current)

		return nil
	},
}

func init() {
	profileCmd.AddCommand(profileCurrentCmd)
}
