package cmd

import (
	"cfmux/internal/profile"
	"cfmux/internal/tunnel"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var tunnelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tunnels in the current profile (merged local + remote)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		current, err := profile.Current()
		if err != nil {
			return err
		}

		res, err := tunnel.List(current)
		if err != nil {
			return err
		}

		if res.RemoteError != nil {
			fmt.Fprintf(os.Stderr, "warning: remote sync failed (%v) — showing local registry only\n\n", res.RemoteError)
		}

		if len(res.Entries) == 0 {
			fmt.Printf("No tunnels in profile %s\n", current)
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tUUID\tSTATE")
		for _, e := range res.Entries {
			fmt.Fprintf(w, "%s\t%s\t%s\n", e.Name, e.UUID, e.State)
		}
		return w.Flush()
	},
}

func init() {
	tunnelCmd.AddCommand(tunnelListCmd)
}
