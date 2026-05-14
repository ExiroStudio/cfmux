package service

import (
	"cfmux/internal/app"
	"errors"
	"fmt"
	"os"
)

// Enable runs `systemctl enable --now <unit>` after verifying the unit file
// exists at the expected location. Output is streamed to the user's terminal.
func Enable(profile, tunnelName string, progress app.ProgressFunc) error {
	if progress == nil {
		progress = func(string) {}
	}

	if err := preflight(PreflightOpts{RequireRoot: true}); err != nil {
		return err
	}

	unitName, err := UnitName(profile, tunnelName)
	if err != nil {
		return err
	}
	unitPath, err := UnitPath(unitName)
	if err != nil {
		return err
	}

	info, err := os.Lstat(unitPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w (looked for %s) — run `cfmux service install %s` first", errNoSuchUnit, unitPath, tunnelName)
		}
		return fmt.Errorf("stat %s: %w", unitPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to enable %s: it is a symlink, not a regular unit file", unitPath)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to enable %s: not a regular file", unitPath)
	}

	progress(fmt.Sprintf("enabling and starting %s", unitName))
	code, err := systemctlInherit("enable", "--now", unitName)
	if err != nil {
		return fmt.Errorf("systemctl enable --now failed: %w", err)
	}
	if code != 0 {
		return &app.ExitError{Code: code, Err: fmt.Errorf("systemctl enable --now exited with code %d", code)}
	}
	return nil
}
