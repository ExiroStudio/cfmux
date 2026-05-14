package cmd

import (
	"cfmux/internal/cloudflared"
	"os"

	"github.com/spf13/cobra"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Manage tunnels",
	Args:  cobra.ArbitraryArgs,
	FParseErrWhitelist: cobra.FParseErrWhitelist{
		UnknownFlags: true,
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		raw := argsAfterTunnel()
		return cloudflared.Execute(append([]string{"tunnel"}, raw...)...)
	},
}

func init() {
	rootCmd.AddCommand(tunnelCmd)
}

// argsAfterTunnel recovers the verbatim args following the "tunnel" token
// in os.Args. Cobra's flag parsing can strip or reorder flags before our
// RunE sees them; for unknown-verb passthrough we need the user's input
// exactly as typed so cloudflared receives every flag.
func argsAfterTunnel() []string {
	for i, a := range os.Args {
		if a == "tunnel" {
			return os.Args[i+1:]
		}
	}
	return nil
}
