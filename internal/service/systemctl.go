package service

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// systemctlRunner executes systemctl with the given argv. It is a package-level
// variable so unit tests can substitute a deterministic fake without ever
// shelling out. Production callers must pass a literal verb as args[0]
// (e.g. "daemon-reload", "stop"); never accept the verb from user input.
//
// Returns:
//   - exitCode: the process exit code (0 on success, 0 if no exit status available)
//   - stderr: captured stderr text (for inclusion in error messages)
//   - err: non-nil if the process did not exit cleanly
var systemctlRunner = realSystemctl

// systemctlInherit runs systemctl with stdin/stdout/stderr connected to the
// current process — used for status, where systemctl's own output is what
// the user wants to see. Tests override it the same way as systemctlRunner.
var systemctlInherit = realSystemctlInherit

func realSystemctl(args ...string) (int, string, error) {
	if len(args) == 0 {
		return 0, "", errors.New("systemctl called with no args")
	}
	cmd := exec.Command("systemctl", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), stderr.String(), err
		}
		return 0, stderr.String(), err
	}
	return 0, stderr.String(), nil
}

func realSystemctlInherit(args ...string) (int, error) {
	if len(args) == 0 {
		return 0, errors.New("systemctl called with no args")
	}
	cmd := exec.Command("systemctl", args...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 0, fmt.Errorf("run systemctl: %w", err)
	}
	return 0, nil
}
