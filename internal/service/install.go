package service

import (
	"cfmux/internal/app"
	"cfmux/internal/tunnel"
	"errors"
	"fmt"
	"os"
)

// Test seams. Production callers never replace these.
var (
	getEUID      = os.Geteuid
	listTunnels  = tunnel.List
	osExecutable = os.Executable
	// resolveBinaryFn is the indirection that tests override when they need
	// Install to accept a binary in a location the production trust-check
	// would reject (e.g. t.TempDir() under /tmp). Production never replaces it.
	resolveBinaryFn = resolveBinary
)

// InstallOpts collects the optional knobs accepted at install time.
type InstallOpts struct {
	// User overrides the SUDO_USER fallback when resolving the unit's `User=`.
	User string
}

// Install renders and writes a cfmux-<profile>-<tunnel>.service unit to
// systemdDir after exhaustive validation. It deliberately does NOT enable
// the service — that's a separate, explicit step.
func Install(profile, tunnelName string, opts InstallOpts, progress app.ProgressFunc) error {
	if progress == nil {
		progress = func(string) {}
	}

	if err := preflight(PreflightOpts{RequireRoot: true}); err != nil {
		return err
	}

	invoker, err := resolveInvokingUser(opts.User)
	if err != nil {
		return err
	}
	progress(fmt.Sprintf("service will run as %s:%s", invoker.Username, invoker.GroupName))

	unitName, err := UnitName(profile, tunnelName)
	if err != nil {
		return err
	}
	unitPath, err := UnitPath(unitName)
	if err != nil {
		return err
	}

	reg, err := tunnel.Load(profile)
	if err != nil {
		return fmt.Errorf("load registry for profile %s: %w", profile, err)
	}
	if _, ok := reg.Find(tunnelName); !ok {
		return fmt.Errorf("tunnel %q is not registered under profile %s", tunnelName, profile)
	}

	progress("verifying tunnel state against Cloudflare")
	res, err := listTunnels(profile)
	if err != nil {
		return fmt.Errorf("list tunnels for profile %s: %w", profile, err)
	}
	if res.RemoteError != nil {
		return fmt.Errorf("cannot verify tunnel against Cloudflare (%w) — connectivity required for service install", res.RemoteError)
	}
	state, ok := findEntryState(res, tunnelName)
	if !ok {
		return fmt.Errorf("tunnel %q not found in merged remote/local list", tunnelName)
	}
	if state != tunnel.StateManaged {
		return fmt.Errorf("refusing to install service for tunnel %q in state %q (must be %q)", tunnelName, state, tunnel.StateManaged)
	}

	credsPath := app.TunnelCredsPath(profile, tunnelName)
	if _, err := os.Stat(credsPath); err != nil {
		return fmt.Errorf("tunnel credentials missing at %s: %w", credsPath, err)
	}
	configPath := app.TunnelConfigPath(profile, tunnelName)
	if _, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("tunnel config missing at %s: %w", configPath, err)
	}

	binary, err := resolveBinaryFn(invoker)
	if err != nil {
		return err
	}
	progress(fmt.Sprintf("ExecStart will be: %s tunnel run %s --profile %s", binary, tunnelName, profile))

	unit, err := RenderUnit(UnitOpts{
		Profile:     profile,
		Tunnel:      tunnelName,
		User:        invoker.Username,
		Group:       invoker.GroupName,
		CfmuxBinary: binary,
	})
	if err != nil {
		return err
	}

	if err := WriteUnit(unitPath, unit); err != nil {
		return err
	}
	progress(fmt.Sprintf("wrote unit file %s", unitPath))

	if code, stderr, err := systemctlRunner("daemon-reload"); err != nil {
		// Roll back the freshly-written unit so we don't leave half-installed state.
		_ = os.Remove(unitPath)
		return fmt.Errorf("systemctl daemon-reload failed (exit %d): %s: %w", code, stderr, err)
	}
	progress("ran systemctl daemon-reload")

	return nil
}

func findEntryState(res tunnel.ListResult, name string) (string, bool) {
	for _, e := range res.Entries {
		if e.Name == name {
			return e.State, true
		}
	}
	return "", false
}

// errNoSuchUnit is returned by status/enable helpers when the unit file does
// not exist at all. Kept simple — callers wrap it with context.
var errNoSuchUnit = errors.New("no cfmux service unit installed for this tunnel")
