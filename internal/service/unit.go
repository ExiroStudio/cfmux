package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// systemdDir is the directory where cfmux installs its unit files.
// It is a package-level variable so tests can redirect writes to t.TempDir().
// Production code never mutates it.
var systemdDir = "/etc/systemd/system"

const unitPrefix = "cfmux-"
const unitSuffix = ".service"

// UnitName composes the systemd unit filename for the (profile, tunnel) pair.
// Both parts are run through sanitizeUnitPart — any rejection here means
// the install will not proceed.
func UnitName(profile, tunnel string) (string, error) {
	p, err := sanitizeUnitPart(profile)
	if err != nil {
		return "", fmt.Errorf("profile: %w", err)
	}
	t, err := sanitizeUnitPart(tunnel)
	if err != nil {
		return "", fmt.Errorf("tunnel: %w", err)
	}
	return unitPrefix + p + "-" + t + unitSuffix, nil
}

// UnitPath returns the absolute filesystem path of the unit file inside
// systemdDir. It performs a defense-in-depth post-check that the cleaned path
// is still strictly inside systemdDir — sanitizeUnitPart should already make
// this impossible, but we verify anyway.
func UnitPath(unitName string) (string, error) {
	if unitName == "" {
		return "", errors.New("unit name is empty")
	}
	if strings.ContainsAny(unitName, "/\\") {
		return "", fmt.Errorf("unit name %q contains a path separator", unitName)
	}
	for _, r := range unitName {
		if r < 0x20 || r == 0x7f {
			return "", fmt.Errorf("unit name %q contains a control character", unitName)
		}
	}
	if strings.Contains(unitName, "..") {
		return "", fmt.Errorf("unit name %q contains \"..\"", unitName)
	}
	if !strings.HasPrefix(unitName, unitPrefix) || !strings.HasSuffix(unitName, unitSuffix) {
		return "", fmt.Errorf("unit name %q does not match cfmux-*.service", unitName)
	}

	root := filepath.Clean(systemdDir)
	abs := filepath.Clean(filepath.Join(root, unitName))
	parent := filepath.Dir(abs)
	if parent != root {
		return "", fmt.Errorf("computed unit path %s escapes %s", abs, root)
	}
	return abs, nil
}

// UnitOpts collects the inputs RenderUnit substitutes into the template.
// All string fields are expected to already be validated by the caller.
type UnitOpts struct {
	Profile     string
	Tunnel      string
	User        string
	Group       string
	CfmuxBinary string
}

// RenderUnit returns the full text of the systemd unit file. It is a pure
// function (no IO, no randomness) so tests can golden-compare its output.
//
// All hardening downgrades from the spec are accompanied by an inline `#`
// comment in the rendered unit so an operator inspecting the file with
// `systemctl cat` understands why they exist.
func RenderUnit(opts UnitOpts) (string, error) {
	if _, err := sanitizeUnitPart(opts.Profile); err != nil {
		return "", fmt.Errorf("profile: %w", err)
	}
	if _, err := sanitizeUnitPart(opts.Tunnel); err != nil {
		return "", fmt.Errorf("tunnel: %w", err)
	}
	if _, err := sanitizeUnitPart(opts.User); err != nil {
		return "", fmt.Errorf("user: %w", err)
	}
	if _, err := sanitizeUnitPart(opts.Group); err != nil {
		return "", fmt.Errorf("group: %w", err)
	}
	if opts.CfmuxBinary == "" || !filepath.IsAbs(opts.CfmuxBinary) {
		return "", fmt.Errorf("cfmux binary path %q must be absolute", opts.CfmuxBinary)
	}
	// Defense-in-depth: the binary path is interpolated into ExecStart. A newline
	// or shell metacharacter here could break parsing of the unit file. The path
	// has been canonicalized via EvalSymlinks/Clean already; reject anything weird.
	if strings.ContainsAny(opts.CfmuxBinary, "\n\r\x00") {
		return "", fmt.Errorf("cfmux binary path contains a control character")
	}

	const tpl = `[Unit]
Description=Cfmux Tunnel %[2]s (profile %[1]s)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=%[3]s
Group=%[4]s
ExecStart=%[5]s tunnel run %[2]s --profile %[1]s
Restart=always
RestartSec=5

# --- security hardening ---
NoNewPrivileges=true
PrivateTmp=true
# ProtectSystem=full (not strict) — strict may block legitimate runtime needs.
# Revisit after cloudflared compatibility validation.
ProtectSystem=full
# ProtectHome=no — cfmux must read ~/.cfmux/<profile>/{cert.pem,configs,tunnels,registry.json}.
# read-only blocks reads on some configurations; tighten later via ReadOnlyPaths=.
ProtectHome=no
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
LockPersonality=true
# MemoryDenyWriteExecute intentionally omitted — Go runtime / cloudflared may mmap RWX pages.
RestrictRealtime=true
RestrictSUIDSGID=true

[Install]
WantedBy=multi-user.target
`
	return fmt.Sprintf(tpl,
		opts.Profile,     // %[1]s
		opts.Tunnel,      // %[2]s
		opts.User,        // %[3]s
		opts.Group,       // %[4]s
		opts.CfmuxBinary, // %[5]s
	), nil
}

// WriteUnit writes `contents` to `absPath` with O_EXCL semantics — if a file
// already exists at that path, the operation fails. The caller is responsible
// for verifying absPath via UnitPath.
func WriteUnit(absPath, contents string) error {
	if !filepath.IsAbs(absPath) {
		return fmt.Errorf("WriteUnit requires absolute path, got %q", absPath)
	}
	f, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create unit file %s: %w", absPath, err)
	}
	defer f.Close()
	if _, err := f.WriteString(contents); err != nil {
		return fmt.Errorf("write unit file %s: %w", absPath, err)
	}
	return nil
}

// RemoveUnit deletes a unit file after verifying it is a regular file
// (not a symlink — defense against an attacker replacing the unit with a
// symlink to a sensitive path and tricking us into removing the wrong file).
func RemoveUnit(absPath string) error {
	if !filepath.IsAbs(absPath) {
		return fmt.Errorf("RemoveUnit requires absolute path, got %q", absPath)
	}
	info, err := os.Lstat(absPath)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to remove %s: it is a symlink, not a regular unit file", absPath)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to remove %s: not a regular file", absPath)
	}
	return os.Remove(absPath)
}
