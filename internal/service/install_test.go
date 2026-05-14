package service

import (
	"cfmux/internal/app"
	"cfmux/internal/tunnel"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
)

// installEnv builds a self-contained, t.TempDir()-rooted CFMUX_HOME, a fake
// cfmux binary in a trusted location, a redirected systemdDir, and seamed
// systemctl/euid/listTunnels — so Install can run end-to-end without root,
// systemctl, or cloudflared.
type installEnv struct {
	t           *testing.T
	home        string
	systemdDir  string
	profile     string
	user        *user.User
	cfmuxBinary string
	calls       []string
}

func newInstallEnv(t *testing.T, profile string) *installEnv {
	t.Helper()

	u, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current: %v", err)
	}
	if u.Uid == "0" {
		t.Skip("running as root; this test exercises the unprivileged path")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", u.Username)

	// Place the fake binary outside HOME and /tmp, to satisfy resolveBinary.
	// We don't actually execute it; it just needs to satisfy the trust checks.
	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(binDir, "cfmux")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	sd := t.TempDir()
	env := &installEnv{
		t:           t,
		home:        home,
		systemdDir:  sd,
		profile:     profile,
		user:        u,
		cfmuxBinary: bin,
	}

	// Test seams — restored at cleanup.
	origEUID := getEUID
	origList := listTunnels
	origRunner := systemctlRunner
	origExec := osExecutable
	origSDir := systemdDir
	origResolveBin := resolveBinaryFn
	getEUID = func() int { return 0 }
	systemctlRunner = func(args ...string) (int, string, error) {
		env.calls = append(env.calls, strings.Join(args, " "))
		return 0, "", nil
	}
	osExecutable = func() (string, error) { return bin, nil }
	// Bypass the binary-trust check (location + parent-mode). This test fixture
	// lives under t.TempDir() (i.e. /tmp on Linux), which the real check rightly
	// rejects. Tests that exercise the trust check explicitly restore the real
	// resolveBinaryFn before calling Install.
	resolveBinaryFn = func(*resolvedUser) (string, error) { return bin, nil }
	systemdDir = sd
	t.Cleanup(func() {
		getEUID = origEUID
		listTunnels = origList
		systemctlRunner = origRunner
		osExecutable = origExec
		systemdDir = origSDir
		resolveBinaryFn = origResolveBin
	})

	// Default: pretend the tunnel exists and is managed.
	listTunnels = func(p string) (tunnel.ListResult, error) {
		return tunnel.ListResult{
			Entries: []tunnel.Entry{{Name: "api", State: tunnel.StateManaged}},
		}, nil
	}

	// Seed an empty profile dir.
	profDir := filepath.Join(home, ".cfmux", "profiles", profile)
	if err := os.MkdirAll(filepath.Join(profDir, "tunnels"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(profDir, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}

	return env
}

func (e *installEnv) addTunnel(name string, withFiles bool) {
	e.t.Helper()
	reg, err := tunnel.Load(e.profile)
	if err != nil {
		e.t.Fatal(err)
	}
	if err := reg.Add(tunnel.Tunnel{
		Name:   name,
		UUID:   "00000000-0000-0000-0000-000000000000",
		Config: "configs/" + name + ".yml",
		Creds:  "tunnels/" + name + ".json",
	}); err != nil {
		e.t.Fatal(err)
	}
	if err := tunnel.Save(e.profile, reg); err != nil {
		e.t.Fatal(err)
	}
	if !withFiles {
		return
	}
	if err := os.WriteFile(app.TunnelCredsPath(e.profile, name), []byte("{}"), 0o600); err != nil {
		e.t.Fatal(err)
	}
	if err := os.WriteFile(app.TunnelConfigPath(e.profile, name), []byte("tunnel: x\n"), 0o644); err != nil {
		e.t.Fatal(err)
	}
}

func TestInstall_HappyPath(t *testing.T) {
	env := newInstallEnv(t, "senvada")
	env.addTunnel("api", true)

	if err := Install(env.profile, "api", InstallOpts{}, nil); err != nil {
		t.Fatalf("Install: %v", err)
	}

	unitPath := filepath.Join(env.systemdDir, "cfmux-senvada-api.service")
	body, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("unit not written: %v", err)
	}
	s := string(body)
	wantExec := "ExecStart=" + env.cfmuxBinary + " tunnel run api --profile senvada"
	if !strings.Contains(s, wantExec) {
		t.Fatalf("unit missing ExecStart line %q\n---\n%s", wantExec, s)
	}
	if !strings.Contains(s, "User="+env.user.Username) {
		t.Fatalf("unit missing User=%s", env.user.Username)
	}

	wantCall := "daemon-reload"
	found := false
	for _, c := range env.calls {
		if c == wantCall {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected systemctl daemon-reload, got: %v", env.calls)
	}
}

func TestInstall_RefusesOverwrite(t *testing.T) {
	env := newInstallEnv(t, "senvada")
	env.addTunnel("api", true)

	if err := Install(env.profile, "api", InstallOpts{}, nil); err != nil {
		t.Fatalf("first install: %v", err)
	}
	err := Install(env.profile, "api", InstallOpts{}, nil)
	if err == nil {
		t.Fatal("second install should fail (O_EXCL)")
	}
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected os.ErrExist, got: %v", err)
	}
}

func TestInstall_RejectsUnknownTunnel(t *testing.T) {
	env := newInstallEnv(t, "senvada")
	// Don't add the tunnel to the registry.

	err := Install(env.profile, "api", InstallOpts{}, nil)
	if err == nil || !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("expected 'not registered' error, got: %v", err)
	}
}

func TestInstall_RejectsMissingCreds(t *testing.T) {
	env := newInstallEnv(t, "senvada")
	env.addTunnel("api", false) // registered but no on-disk files
	// Recreate config so only creds are missing.
	if err := os.WriteFile(app.TunnelConfigPath(env.profile, "api"), []byte("tunnel: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Install(env.profile, "api", InstallOpts{}, nil)
	if err == nil || !strings.Contains(err.Error(), "credentials missing") {
		t.Fatalf("expected credentials-missing error, got: %v", err)
	}
}

func TestInstall_RejectsMissingConfig(t *testing.T) {
	env := newInstallEnv(t, "senvada")
	env.addTunnel("api", false)
	// Recreate creds so only config is missing.
	if err := os.WriteFile(app.TunnelCredsPath(env.profile, "api"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := Install(env.profile, "api", InstallOpts{}, nil)
	if err == nil || !strings.Contains(err.Error(), "config missing") {
		t.Fatalf("expected config-missing error, got: %v", err)
	}
}

func TestInstall_RejectsStaleState(t *testing.T) {
	env := newInstallEnv(t, "senvada")
	env.addTunnel("api", true)
	listTunnels = func(p string) (tunnel.ListResult, error) {
		return tunnel.ListResult{
			Entries: []tunnel.Entry{{Name: "api", State: tunnel.StateStale}},
		}, nil
	}

	err := Install(env.profile, "api", InstallOpts{}, nil)
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("expected stale-state error, got: %v", err)
	}
}

func TestInstall_RejectsRemoteError(t *testing.T) {
	env := newInstallEnv(t, "senvada")
	env.addTunnel("api", true)
	listTunnels = func(p string) (tunnel.ListResult, error) {
		return tunnel.ListResult{
			Entries:     []tunnel.Entry{{Name: "api", State: tunnel.StateManaged}},
			RemoteError: errors.New("cloudflared not in PATH"),
		}, nil
	}

	err := Install(env.profile, "api", InstallOpts{}, nil)
	if err == nil || !strings.Contains(err.Error(), "Cloudflare") {
		t.Fatalf("expected Cloudflare-verification error, got: %v", err)
	}
}

func TestInstall_RejectsBinaryInTmp(t *testing.T) {
	env := newInstallEnv(t, "senvada")
	env.addTunnel("api", true)

	// This test exercises the real trust check, so restore the production
	// resolveBinary that newInstallEnv stubbed out.
	resolveBinaryFn = resolveBinary

	// Place the fake binary directly in /tmp and point osExecutable at it.
	tmpBin := filepath.Join(os.TempDir(), "cfmux-test-binary")
	if err := os.WriteFile(tmpBin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(tmpBin) })
	osExecutable = func() (string, error) { return tmpBin, nil }

	err := Install(env.profile, "api", InstallOpts{}, nil)
	if err == nil || !strings.Contains(err.Error(), "user-writable") {
		t.Fatalf("expected user-writable-location error, got: %v", err)
	}
}

func TestStatus_PropagatesExitCode(t *testing.T) {
	t.Setenv("PATH", os.Getenv("PATH")) // keep systemctl-lookup happy on test host
	// Skip if systemctl isn't actually available — preflight will reject.
	if _, err := os.Stat("/usr/bin/systemctl"); err != nil {
		if _, err := os.Stat("/bin/systemctl"); err != nil {
			t.Skip("systemctl not available")
		}
	}

	origInherit := systemctlInherit
	systemctlInherit = func(args ...string) (int, error) { return 3, nil }
	t.Cleanup(func() { systemctlInherit = origInherit })

	err := Status("senvada", "api")
	if err == nil {
		t.Fatal("Status should return an ExitError for non-zero codes")
	}
	var exitErr *app.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *app.ExitError, got %T: %v", err, err)
	}
	if exitErr.Code != 3 {
		t.Fatalf("expected code 3, got %d", exitErr.Code)
	}
}
