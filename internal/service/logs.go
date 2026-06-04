package service

import (
	"cfmux/internal/app"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// execLookPath is a test seam for confirming journalctl is installed. Logs
// uses its own lookup rather than preflight() because preflight checks for
// systemctl, and logs only needs journalctl.
var execLookPath = exec.LookPath

// journalctlRun executes journalctl with stdin/stdout/stderr connected to the
// current process — the user wants to read (or follow) the live journal, not
// have cfmux buffer it. This mirrors how Status streams systemctl. It is a
// package-level variable so tests can assert on the argv without exec'ing.
var journalctlRun = realJournalctl

func realJournalctl(args ...string) (int, error) {
	cmd := exec.Command("journalctl", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 0, fmt.Errorf("run journalctl: %w", err)
	}
	return 0, nil
}

// LogsOpts collects the journalctl knobs cfmux exposes. Intentionally minimal:
// power users who need --since/--grep/--priority already know `journalctl -u`.
type LogsOpts struct {
	// Follow maps to journalctl --follow (-f).
	Follow bool
	// Lines maps to journalctl -n; 0 means leave journalctl's default.
	Lines int
}

// Logs streams `journalctl -u <unit>` for a tunnel's service. No root is
// required — journald grants read access for system units on modern systemd.
// If the system is locked down, journalctl fails naturally with a permission
// error, which we propagate.
func Logs(profile, tunnelName string, opts LogsOpts) error {
	if _, err := execLookPath("journalctl"); err != nil {
		return errors.New("journalctl not found in PATH — cfmux service logs requires systemd's journald")
	}

	unitName, err := UnitName(profile, tunnelName)
	if err != nil {
		return err
	}

	args := []string{"-u", unitName}
	if opts.Lines > 0 {
		args = append(args, "-n", strconv.Itoa(opts.Lines))
	}
	if opts.Follow {
		args = append(args, "--follow")
	}

	code, err := journalctlRun(args...)
	if err != nil {
		return err
	}
	if code != 0 {
		return &app.ExitError{Code: code}
	}
	return nil
}
