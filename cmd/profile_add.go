package cmd

import (
	"cfmux/internal/profile"
	"fmt"

	"github.com/spf13/cobra"
)

var profileAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add new profile",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		if err := profile.Add(args[0], printProgress); err != nil {
			return err
		}
		fmt.Println()
		fmt.Printf("Profile %s added successfully\n", args[0])
		fmt.Printf("Now you can use this profile by running: cfmux profile use %s\n", args[0])
		return nil
	},
}

func init() {
	profileCmd.AddCommand(profileAddCmd)
}
