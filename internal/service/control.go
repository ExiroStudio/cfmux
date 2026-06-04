package service

import (
	"cfmux/internal/app"
	"fmt"
	"strings"
)

// controlActions is the allow-list of lifecycle verbs Control accepts, mapped
// to the past-tense word used in the success progress line. Keeping it as a
// package-level set (rather than an inline switch) means the entry point can
// reject anything outside the list before touching systemd, and the verb is
// never taken verbatim from user input.
var controlActions = map[string]string{
	"start":   "started",
	"stop":    "stopped",
	"restart": "restarted",
	"disable": "disabled",
}

// Control runs `systemctl <action> <unit>` for a lifecycle mutation
// (start/stop/restart/disable). All four require root because they mutate unit
// state or, for disable, the enable symlinks under /etc/systemd/system/*.wants/.
//
// Unlike enable/install/uninstall, Control does NOT pre-validate the unit file
// on disk (no Lstat, no symlink rejection). systemctl is the authority on what
// units exist — it knows about generators and transient units that never live
// in /etc/systemd/system. We only sanitize the (profile, tunnel) pair via
// UnitName and let systemctl report unit-not-found itself, wrapping the common
// exit-5 (unit not loaded) case with an install hint.
//
// Output uses systemctlRunner (captured stderr): these verbs are terse and
// cfmux owns the UX via progress messages, unlike status which streams.
func Control(profile, tunnelName, action string, progress app.ProgressFunc) error {
	if progress == nil {
		progress = func(string) {}
	}

	pastTense, ok := controlActions[action]
	if !ok {
		return fmt.Errorf("unsupported service action %q", action)
	}

	if err := preflight(PreflightOpts{RequireRoot: true}); err != nil {
		return err
	}

	unitName, err := UnitName(profile, tunnelName)
	if err != nil {
		return err
	}

	progress(fmt.Sprintf("running systemctl %s %s", action, unitName))
	code, stderr, err := systemctlRunner(action, unitName)
	if err == nil {
		progress(fmt.Sprintf("%s %s", pastTense, unitName))
		return nil
	}

	// code == 0 with a non-nil err means systemctl never exited cleanly
	// (e.g. binary missing) — not a meaningful systemctl exit code.
	if code == 0 {
		return fmt.Errorf("run systemctl %s: %w", action, err)
	}
	// Exit 5 = unit not loaded. Mirror Status()'s exit-4 hint.
	if code == 5 {
		return &app.ExitError{Code: 5, Err: fmt.Errorf("no systemd unit named %s is loaded — run `cfmux service install %s` first", unitName, tunnelName)}
	}
	msg := strings.TrimSpace(stderr)
	if msg == "" {
		msg = fmt.Sprintf("systemctl %s exited with code %d", action, code)
	}
	return &app.ExitError{Code: code, Err: fmt.Errorf("systemctl %s %s failed: %s", action, unitName, msg)}
}

// IsEnabled wraps `systemctl is-enabled <unit>`. It is a read-only query and
// requires no root, matching Status(). It prints "enabled" / "disabled" and
// propagates systemctl's exit code so scripts can branch on it.
func IsEnabled(profile, tunnelName string) error {
	return query(profile, tunnelName, "is-enabled", "enabled", "disabled")
}

// IsActive wraps `systemctl is-active <unit>`. Read-only, no root required.
// Prints "active" / "inactive" and propagates the exit code (0=active,
// 3=inactive, ...).
func IsActive(profile, tunnelName string) error {
	return query(profile, tunnelName, "is-active", "active", "inactive")
}

// query is the shared body of IsEnabled/IsActive. systemctl's own stdout is
// discarded (systemctlRunner only captures stderr); we print our own
// human-readable line keyed off the exit code, matching the behaviour of
// `systemctl is-active`/`is-enabled` themselves.
func query(profile, tunnelName, verb, trueWord, falseWord string) error {
	if err := preflight(PreflightOpts{RequireRoot: false}); err != nil {
		return err
	}

	unitName, err := UnitName(profile, tunnelName)
	if err != nil {
		return err
	}

	code, _, err := systemctlRunner(verb, unitName)
	if err == nil {
		fmt.Println(trueWord)
		return nil
	}
	if code == 0 {
		return fmt.Errorf("run systemctl %s: %w", verb, err)
	}
	fmt.Println(falseWord)
	return &app.ExitError{Code: code}
}
