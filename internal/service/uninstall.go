package service

import (
	"cfmux/internal/app"
	"errors"
	"fmt"
	"os"
)

// Uninstall stops + disables + removes the unit file for (profile, tunnel),
// then runs daemon-reload. It deliberately leaves the cfmux registry, configs,
// and credentials untouched — those are owned by `cfmux tunnel delete`.
//
// Stop and disable are best-effort: if the unit is already inactive or
// already disabled, we still proceed. The unit file removal is the
// non-negotiable step.
func Uninstall(profile, tunnelName string, progress app.ProgressFunc) error {
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

	info, statErr := os.Lstat(unitPath)
	if statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return fmt.Errorf("%w (looked for %s)", errNoSuchUnit, unitPath)
		}
		return fmt.Errorf("stat %s: %w", unitPath, statErr)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to uninstall %s: it is a symlink, not a regular unit file", unitPath)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to uninstall %s: not a regular file", unitPath)
	}

	// Best-effort stop.
	if code, stderr, err := systemctlRunner("stop", unitName); err != nil {
		progress(fmt.Sprintf("systemctl stop reported exit %d (continuing): %s", code, stderr))
	} else {
		progress(fmt.Sprintf("stopped %s", unitName))
	}

	// Best-effort disable.
	if code, stderr, err := systemctlRunner("disable", unitName); err != nil {
		progress(fmt.Sprintf("systemctl disable reported exit %d (continuing): %s", code, stderr))
	} else {
		progress(fmt.Sprintf("disabled %s", unitName))
	}

	if err := RemoveUnit(unitPath); err != nil {
		return fmt.Errorf("remove unit %s: %w", unitPath, err)
	}
	progress(fmt.Sprintf("removed %s", unitPath))

	if code, stderr, err := systemctlRunner("daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload failed (exit %d): %s: %w", code, stderr, err)
	}
	progress("ran systemctl daemon-reload")

	return nil
}
