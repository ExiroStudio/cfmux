package cmd

import (
	"cfmux/internal/profile"
	"cfmux/internal/service"

	"github.com/spf13/cobra"
)

var (
	logsFollow bool
	logsLines  int
)

var serviceLogsCmd = &cobra.Command{
	Use:   "logs [tunnel]",
	Short: "Show journald logs for the tunnel's service (no sudo required)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		current, err := profile.Current()
		if err != nil {
			return err
		}
		return service.Logs(current, args[0], service.LogsOpts{
			Follow: logsFollow,
			Lines:  logsLines,
		})
	},
}

func init() {
	serviceLogsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "follow the journal (like tail -f)")
	serviceLogsCmd.Flags().IntVarP(&logsLines, "lines", "n", 0, "number of journal lines to show (0 = journalctl default)")
	serviceCmd.AddCommand(serviceLogsCmd)
}
