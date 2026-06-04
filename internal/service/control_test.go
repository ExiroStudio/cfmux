package service

import (
	"cfmux/internal/app"
	"errors"
	"strings"
	"testing"
)

// fakeSystemctl installs a systemctlRunner stub that records the argv of each
// call and returns the supplied (code, stderr, err). It also forces euid to
// root so RequireRoot:true preflights pass, and restores both at cleanup.
func fakeSystemctl(t *testing.T, code int, stderr string, runErr error) *[]string {
	t.Helper()
	calls := &[]string{}
	origRunner := systemctlRunner
	origEUID := getEUID
	getEUID = func() int { return 0 }
	systemctlRunner = func(args ...string) (int, string, error) {
		*calls = append(*calls, strings.Join(args, " "))
		return code, stderr, runErr
	}
	t.Cleanup(func() {
		systemctlRunner = origRunner
		getEUID = origEUID
	})
	return calls
}

func TestControl_Start_HappyPath(t *testing.T) {
	calls := fakeSystemctl(t, 0, "", nil)
	var progress []string

	if err := Control("senvada", "api", "start", func(m string) { progress = append(progress, m) }); err != nil {
		t.Fatalf("Control start: %v", err)
	}
	if len(*calls) != 1 || (*calls)[0] != "start cfmux-senvada-api.service" {
		t.Fatalf("unexpected systemctl calls: %v", *calls)
	}
	if len(progress) == 0 || !strings.Contains(progress[len(progress)-1], "started cfmux-senvada-api.service") {
		t.Fatalf("expected success progress, got %v", progress)
	}
}

func TestControl_Stop_HappyPath(t *testing.T) {
	calls := fakeSystemctl(t, 0, "", nil)
	if err := Control("senvada", "api", "stop", nil); err != nil {
		t.Fatalf("Control stop: %v", err)
	}
	if (*calls)[0] != "stop cfmux-senvada-api.service" {
		t.Fatalf("unexpected call: %v", *calls)
	}
}

func TestControl_Restart_HappyPath(t *testing.T) {
	calls := fakeSystemctl(t, 0, "", nil)
	if err := Control("senvada", "api", "restart", nil); err != nil {
		t.Fatalf("Control restart: %v", err)
	}
	if (*calls)[0] != "restart cfmux-senvada-api.service" {
		t.Fatalf("unexpected call: %v", *calls)
	}
}

func TestControl_Disable_HappyPath(t *testing.T) {
	calls := fakeSystemctl(t, 0, "", nil)
	if err := Control("senvada", "api", "disable", nil); err != nil {
		t.Fatalf("Control disable: %v", err)
	}
	if (*calls)[0] != "disable cfmux-senvada-api.service" {
		t.Fatalf("unexpected call: %v", *calls)
	}
}

func TestControl_RejectsInvalidAction(t *testing.T) {
	calls := fakeSystemctl(t, 0, "", nil)
	err := Control("senvada", "api", "frobnicate", nil)
	if err == nil {
		t.Fatal("expected error for action outside allow-list")
	}
	if len(*calls) != 0 {
		t.Fatalf("systemctl must not be called for invalid action, got %v", *calls)
	}
}

func TestControl_PropagatesExitCode(t *testing.T) {
	fakeSystemctl(t, 1, "Job failed", errors.New("exit status 1"))
	err := Control("senvada", "api", "restart", nil)
	var exitErr *app.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *app.ExitError, got %T: %v", err, err)
	}
	if exitErr.Code != 1 {
		t.Fatalf("expected code 1, got %d", exitErr.Code)
	}
}

func TestControl_ExitCode5_HintsInstall(t *testing.T) {
	fakeSystemctl(t, 5, "Unit cfmux-senvada-api.service not loaded.", errors.New("exit status 5"))
	err := Control("senvada", "api", "start", nil)
	if err == nil || !strings.Contains(err.Error(), "install") {
		t.Fatalf("expected exit-5 error to hint install, got %v", err)
	}
	var exitErr *app.ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 5 {
		t.Fatalf("expected ExitError code 5, got %v", err)
	}
}

func TestControl_BadTunnelName(t *testing.T) {
	calls := fakeSystemctl(t, 0, "", nil)
	err := Control("senvada", "a;rm -rf /", "start", nil)
	if err == nil {
		t.Fatal("expected sanitization error for malicious tunnel name")
	}
	if len(*calls) != 0 {
		t.Fatalf("systemctl must not be called when sanitization fails, got %v", *calls)
	}
}

func TestIsEnabled_True(t *testing.T) {
	fakeSystemctl(t, 0, "", nil)
	if err := IsEnabled("senvada", "api"); err != nil {
		t.Fatalf("IsEnabled true: %v", err)
	}
}

func TestIsEnabled_False(t *testing.T) {
	fakeSystemctl(t, 1, "", errors.New("exit status 1"))
	err := IsEnabled("senvada", "api")
	var exitErr *app.ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("expected ExitError code 1, got %v", err)
	}
}

func TestIsActive_True(t *testing.T) {
	fakeSystemctl(t, 0, "", nil)
	if err := IsActive("senvada", "api"); err != nil {
		t.Fatalf("IsActive true: %v", err)
	}
}

func TestIsActive_False(t *testing.T) {
	fakeSystemctl(t, 3, "", errors.New("exit status 3"))
	err := IsActive("senvada", "api")
	var exitErr *app.ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 3 {
		t.Fatalf("expected ExitError code 3, got %v", err)
	}
}

func TestIsActive_NoRoot(t *testing.T) {
	// Queries must not require root. Force a non-root euid and confirm the
	// query reaches systemctl rather than being rejected by preflight.
	origRunner := systemctlRunner
	origEUID := getEUID
	getEUID = func() int { return 1000 }
	called := false
	systemctlRunner = func(args ...string) (int, string, error) {
		called = true
		return 0, "", nil
	}
	t.Cleanup(func() {
		systemctlRunner = origRunner
		getEUID = origEUID
	})

	if err := IsActive("senvada", "api"); err != nil {
		t.Fatalf("IsActive should not require root: %v", err)
	}
	if !called {
		t.Fatal("expected systemctl to be invoked for non-root query")
	}
}
