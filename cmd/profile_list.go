package cmd

import (
	"cfmux/internal/profile"
	"fmt"

	"github.com/spf13/cobra"
)

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List profiles",

	RunE: func(cmd *cobra.Command, args []string) error {
		profiles, err := profile.List()
		if err != nil {
			return err
		}

		current, _ := profile.Current()

		fmt.Println("Profiles:")
		for _, p := range profiles {
			prefix := " "

			if p == current {
				prefix = "→"
			}

			fmt.Printf("%s %s\n", prefix, p)
		}

		return nil
	},
}

func init() {
	profileCmd.AddCommand(profileListCmd)
}
