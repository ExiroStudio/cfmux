package service

import (
	"os/user"
	"strings"
	"testing"
)

func TestSanitizeUnitPart_AcceptsValid(t *testing.T) {
	cases := []string{
		"api",
		"a",
		"A",
		"0",
		"api-v2",
		"web_3",
		"My-Tunnel_42",
		strings.Repeat("a", maxUnitPartLen),
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			got, err := sanitizeUnitPart(c)
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", c, err)
			}
			if got != c {
				t.Fatalf("sanitizeUnitPart should not modify input: got %q want %q", got, c)
			}
		})
	}
}

func TestSanitizeUnitPart_RejectsMalicious(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"path_traversal", "../etc/passwd"},
		{"forward_slash", "a/b"},
		{"backslash", "a\\b"},
		{"space", "a b"},
		{"semicolon_injection", "a;rm -rf /"},
		{"newline", "a\nb"},
		{"cr", "a\rb"},
		{"dollar_expansion", "a$b"},
		{"backtick", "a`b`c"},
		{"nul_byte", "a\x00b"},
		{"leading_dash", "-leading"},
		{"trailing_dash", "trailing-"},
		{"leading_underscore", "_leading"},
		{"trailing_underscore", "trailing_"},
		{"unicode_letter", "café"},
		{"unicode_homoglyph", "арi"},
		{"too_long", strings.Repeat("a", maxUnitPartLen+1)},
		{"dot", "a.b"},
		{"at_sign", "user@host"},
		{"quote", "a\"b"},
		{"single_quote", "a'b"},
		{"angle", "a<b"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := sanitizeUnitPart(c.input)
			if err == nil {
				t.Fatalf("sanitizeUnitPart(%q) should have failed", c.input)
			}
		})
	}
}

func TestResolveInvokingUser_FlagBeatsSudoUser(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current: %v", err)
	}
	if u.Uid == "0" {
		t.Skip("running as root — cannot test non-root user resolution")
	}

	t.Setenv("SUDO_USER", "definitely-not-a-real-user-xyzzy")

	got, err := resolveInvokingUser(u.Username)
	if err != nil {
		t.Fatalf("resolveInvokingUser(flag=%q) returned unexpected error: %v", u.Username, err)
	}
	if got.Username != u.Username {
		t.Fatalf("flag did not take precedence: got %q, want %q", got.Username, u.Username)
	}
}

func TestResolveInvokingUser_FallsBackToSudoUser(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current: %v", err)
	}
	if u.Uid == "0" {
		t.Skip("running as root — SUDO_USER fallback only applies in sudo context")
	}

	// Simulate running as root via sudo so the SUDO_USER branch is entered.
	origEUID := getEUID
	getEUID = func() int { return 0 }
	t.Cleanup(func() { getEUID = origEUID })

	t.Setenv("SUDO_USER", u.Username)
	got, err := resolveInvokingUser("")
	if err != nil {
		t.Fatalf("resolveInvokingUser(SUDO_USER=%q) returned unexpected error: %v", u.Username, err)
	}
	if got.Username != u.Username {
		t.Fatalf("SUDO_USER not honoured: got %q, want %q", got.Username, u.Username)
	}
}

func TestResolveInvokingUser_FallsBackToCurrentUser(t *testing.T) {
	t.Setenv("SUDO_USER", "")
	u, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current: %v", err)
	}
	got, err := resolveInvokingUser("")
	if err != nil {
		t.Fatalf("resolveInvokingUser should fall back to current user, got error: %v", err)
	}
	if got.Username != u.Username {
		t.Fatalf("expected current user %q, got %q", u.Username, got.Username)
	}
}

func TestResolveInvokingUser_AllowsRoot(t *testing.T) {
	// Root runtime is intentionally valid for VPS/system-level deployments.
	got, err := resolveInvokingUser("root")
	if err != nil {
		t.Fatalf("resolveInvokingUser(root) unexpected error: %v", err)
	}
	if got.Username != "root" {
		t.Fatalf("expected username root, got %q", got.Username)
	}
	if got.UID != 0 {
		t.Fatalf("expected UID 0, got %d", got.UID)
	}
}

func TestResolveInvokingUser_RejectsInjectionInUsername(t *testing.T) {
	if _, err := resolveInvokingUser("alice;rm -rf /"); err == nil {
		t.Fatal("resolveInvokingUser should refuse usernames containing illegal characters")
	}
}

func TestPreflight_NoPanicOnMissingSystemctl(t *testing.T) {
	t.Setenv("PATH", "")
	err := preflight(PreflightOpts{RequireRoot: false})
	if err == nil {
		t.Fatal("preflight with empty PATH should error")
	}
	if !strings.Contains(err.Error(), "systemctl") {
		t.Fatalf("expected systemctl-related error, got: %v", err)
	}
}

func TestPathHasPrefix(t *testing.T) {
	cases := []struct {
		path, prefix string
		want         bool
	}{
		{"/foo", "/foo", true},
		{"/foo/bar", "/foo", true},
		{"/foobar", "/foo", false},
		{"/foo/../foo/bar", "/foo", true},
		{"/usr/local/bin/cfmux", "/tmp", false},
		{"/tmp/cfmux", "/tmp", true},
		{"/var/tmp/cfmux", "/var/tmp", true},
	}
	for _, c := range cases {
		t.Run(c.path+"_under_"+c.prefix, func(t *testing.T) {
			if got := pathHasPrefix(c.path, c.prefix); got != c.want {
				t.Fatalf("pathHasPrefix(%q,%q) = %v, want %v", c.path, c.prefix, got, c.want)
			}
		})
	}
}
