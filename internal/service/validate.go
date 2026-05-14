package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

const maxUnitPartLen = 64

// sanitizeUnitPart enforces a strict character set on profile / tunnel names
// that become part of a systemd unit filename or are interpolated into a
// rendered unit. It rejects rather than normalizes — any silent transform
// here would hide a bug.
//
// Allowed: [A-Za-z0-9_-]. First and last rune must be alphanumeric.
// Length: 1..maxUnitPartLen.
func sanitizeUnitPart(s string) (string, error) {
	if s == "" {
		return "", errors.New("name is empty")
	}
	if len(s) > maxUnitPartLen {
		return "", fmt.Errorf("name %q exceeds %d characters", s, maxUnitPartLen)
	}

	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return "", fmt.Errorf("name %q contains illegal character at position %d", s, i)
		}
	}

	first := rune(s[0])
	last := rune(s[len(s)-1])
	if !isAlnum(first) {
		return "", fmt.Errorf("name %q must start with an alphanumeric character", s)
	}
	if !isAlnum(last) {
		return "", fmt.Errorf("name %q must end with an alphanumeric character", s)
	}

	return s, nil
}

func isAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// PreflightOpts toggles privilege requirements per command.
type PreflightOpts struct {
	RequireRoot bool
}

// preflight verifies the environment before any systemd interaction:
// linux only, systemctl present, optionally root.
func preflight(opts PreflightOpts) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("cfmux service requires linux (current OS: %s)", runtime.GOOS)
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return errors.New("systemctl not found in PATH — cfmux service requires systemd")
	}
	if opts.RequireRoot && getEUID() != 0 {
		return errors.New("this command requires root — please re-run with sudo")
	}
	return nil
}

// resolvedUser holds the validated identity that will own the systemd service.
type resolvedUser struct {
	Username  string
	GroupName string
	HomeDir   string
	UID       uint32
}

// resolveInvokingUser determines the unprivileged user the service will run as.
//
// Precedence (most to least trusted):
//  1. explicit --user flag
//  2. SUDO_USER env var (only as a fallback — empty under su, unreliable in
//     containers, may be wrong under nested sudo)
//
// If neither yields a value, callers running as root must error out: there
// is no safe default.
func resolveInvokingUser(flagUser string) (*resolvedUser, error) {
	name := strings.TrimSpace(flagUser)
	if name == "" {
		name = strings.TrimSpace(os.Getenv("SUDO_USER"))
	}
	if name == "" {
		return nil, errors.New("could not determine invoking user; pass --user <name> (SUDO_USER is empty under su / nested sudo / containers)")
	}
	if _, err := sanitizeUnitPart(name); err != nil {
		return nil, fmt.Errorf("invalid user name: %w", err)
	}

	u, err := user.Lookup(name)
	if err != nil {
		return nil, fmt.Errorf("lookup user %q: %w", name, err)
	}
	g, err := user.LookupGroupId(u.Gid)
	if err != nil {
		return nil, fmt.Errorf("lookup primary group for %q: %w", name, err)
	}
	if _, err := sanitizeUnitPart(g.Name); err != nil {
		return nil, fmt.Errorf("invalid group name: %w", err)
	}

	uid, err := parseUint32(u.Uid)
	if err != nil {
		return nil, fmt.Errorf("parse uid for %q: %w", name, err)
	}
	if uid == 0 {
		return nil, fmt.Errorf("refusing to install a service to run as root (user %q has uid 0)", name)
	}

	return &resolvedUser{
		Username:  u.Username,
		GroupName: g.Name,
		HomeDir:   u.HomeDir,
		UID:       uid,
	}, nil
}

func parseUint32(s string) (uint32, error) {
	if s == "" {
		return 0, errors.New("empty")
	}
	var n uint64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("non-digit %q", r)
		}
		n = n*10 + uint64(r-'0')
		if n > 1<<32-1 {
			return 0, errors.New("overflow")
		}
	}
	return uint32(n), nil
}

// resolveBinary returns a canonical absolute path to the cfmux binary,
// after multiple defensive checks. We refuse to install a privileged service
// pointing at a binary in any user-writable location.
func resolveBinary(invoker *resolvedUser) (string, error) {
	raw, err := osExecutable()
	if err != nil {
		return "", fmt.Errorf("locate cfmux binary: %w", err)
	}
	abs, err := filepath.EvalSymlinks(raw)
	if err != nil {
		return "", fmt.Errorf("resolve symlinks for %s: %w", raw, err)
	}
	abs = filepath.Clean(abs)
	if !filepath.IsAbs(abs) {
		return "", fmt.Errorf("resolved cfmux path %q is not absolute", abs)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", abs, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", abs)
	}
	if info.Mode().Perm()&0o002 != 0 {
		return "", fmt.Errorf("%s is world-writable — refusing to install a privileged service pointing at it", abs)
	}
	if err := checkOwnership(info, invoker); err != nil {
		return "", fmt.Errorf("%s: %w", abs, err)
	}

	// Location check first — more specific, clearer error than the generic
	// parent-permissions check that follows.
	if err := checkBinaryLocation(abs, invoker); err != nil {
		return "", err
	}

	parent := filepath.Dir(abs)
	pinfo, err := os.Stat(parent)
	if err != nil {
		return "", fmt.Errorf("stat parent %s: %w", parent, err)
	}
	if pinfo.Mode().Perm()&0o002 != 0 {
		return "", fmt.Errorf("parent directory %s is world-writable — refusing", parent)
	}

	return abs, nil
}

// checkBinaryLocation rejects cfmux binaries living in known user-writable
// or ephemeral locations.
func checkBinaryLocation(abs string, invoker *resolvedUser) error {
	forbidden := []string{"/tmp", "/var/tmp", "/dev/shm"}
	if invoker != nil && invoker.HomeDir != "" {
		// Clean the home dir before comparing — defends against `..` in /etc/passwd.
		forbidden = append(forbidden, filepath.Clean(invoker.HomeDir))
	}
	for _, root := range forbidden {
		if pathHasPrefix(abs, root) {
			return fmt.Errorf("cfmux binary at %s lives under %s — refusing to install a privileged service from a user-writable location", abs, root)
		}
	}
	return nil
}

// pathHasPrefix returns true if `path` is exactly `prefix` or a descendant of it.
// It avoids the classic `/foo` vs `/foobar` substring trap by checking for a
// trailing separator after the prefix.
func pathHasPrefix(path, prefix string) bool {
	prefix = filepath.Clean(prefix)
	path = filepath.Clean(path)
	if path == prefix {
		return true
	}
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	return path[len(prefix)] == filepath.Separator
}
