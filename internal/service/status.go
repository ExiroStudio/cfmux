package service

import (
	"cfmux/internal/app"
	"fmt"
)

// Status wraps `systemctl status <unit>`. stdout/stderr stream to the user.
// systemctl's exit code is propagated via app.ExitError so scripts can
// branch on the result (0=active, 3=inactive, 4=not-loaded, ...).
//
// No root required — `systemctl status` works for any user on system units.
func Status(profile, tunnelName string) error {
	if err := preflight(PreflightOpts{RequireRoot: false}); err != nil {
		return err
	}

	unitName, err := UnitName(profile, tunnelName)
	if err != nil {
		return err
	}

	code, err := systemctlInherit("status", unitName)
	if err != nil {
		return fmt.Errorf("run systemctl status: %w", err)
	}
	if code == 0 {
		return nil
	}
	// Friendlier message on the very common "no such unit" case (exit 4).
	if code == 4 {
		return &app.ExitError{Code: 4, Err: fmt.Errorf("no systemd unit named %s — install it with `cfmux service install %s`", unitName, tunnelName)}
	}
	// Other non-zero codes (3 = inactive, 1 = failed, etc.) are normal status
	// information — propagate the code without a synthetic error message.
	return &app.ExitError{Code: code}
}
