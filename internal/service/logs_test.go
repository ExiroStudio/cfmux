package service

import (
	"errors"
	"strings"
	"testing"
)

// fakeJournalctl installs a journalctlRun stub that records the argv (after the
// "journalctl" binary name) and returns (0, nil), with execLookPath forced to
// succeed. Both are restored at cleanup.
func fakeJournalctl(t *testing.T) *[]string {
	t.Helper()
	calls := &[]string{}
	origRun := journalctlRun
	origLook := execLookPath
	execLookPath = func(string) (string, error) { return "/usr/bin/journalctl", nil }
	journalctlRun = func(args ...string) (int, error) {
		*calls = append(*calls, strings.Join(args, " "))
		return 0, nil
	}
	t.Cleanup(func() {
		journalctlRun = origRun
		execLookPath = origLook
	})
	return calls
}

func TestLogs_DefaultArgs(t *testing.T) {
	calls := fakeJournalctl(t)
	if err := Logs("senvada", "api", LogsOpts{}); err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if len(*calls) != 1 || (*calls)[0] != "-u cfmux-senvada-api.service" {
		t.Fatalf("unexpected journalctl argv: %v", *calls)
	}
}

func TestLogs_Follow(t *testing.T) {
	calls := fakeJournalctl(t)
	if err := Logs("senvada", "api", LogsOpts{Follow: true}); err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if !strings.Contains((*calls)[0], "--follow") {
		t.Fatalf("expected --follow in argv: %v", *calls)
	}
}

func TestLogs_Lines(t *testing.T) {
	calls := fakeJournalctl(t)
	if err := Logs("senvada", "api", LogsOpts{Lines: 50}); err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if !strings.Contains((*calls)[0], "-n 50") {
		t.Fatalf("expected -n 50 in argv: %v", *calls)
	}
}

func TestLogs_FollowAndLines(t *testing.T) {
	calls := fakeJournalctl(t)
	if err := Logs("senvada", "api", LogsOpts{Follow: true, Lines: 20}); err != nil {
		t.Fatalf("Logs: %v", err)
	}
	got := (*calls)[0]
	if !strings.Contains(got, "-n 20") || !strings.Contains(got, "--follow") {
		t.Fatalf("expected both -n 20 and --follow: %v", got)
	}
}

func TestLogs_BadTunnelName(t *testing.T) {
	calls := fakeJournalctl(t)
	err := Logs("senvada", "a;rm -rf /", LogsOpts{})
	if err == nil {
		t.Fatal("expected sanitization error for malicious tunnel name")
	}
	if len(*calls) != 0 {
		t.Fatalf("journalctl must not be invoked when sanitization fails: %v", *calls)
	}
}

func TestLogs_MissingJournalctl(t *testing.T) {
	origLook := execLookPath
	execLookPath = func(string) (string, error) { return "", errors.New("not found") }
	t.Cleanup(func() { execLookPath = origLook })

	err := Logs("senvada", "api", LogsOpts{})
	if err == nil || !strings.Contains(err.Error(), "journalctl") {
		t.Fatalf("expected clear journalctl-missing error, got %v", err)
	}
}
