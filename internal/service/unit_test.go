package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnitName_Composes(t *testing.T) {
	got, err := UnitName("senvada", "api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "cfmux-senvada-api.service"
	if got != want {
		t.Fatalf("UnitName = %q, want %q", got, want)
	}
}

func TestUnitName_RejectsInjection(t *testing.T) {
	cases := []struct{ profile, tunnel string }{
		{"senvada", "../etc/passwd"},
		{"sen/vada", "api"},
		{"senvada", "api;rm -rf /"},
		{"", "api"},
		{"senvada", ""},
		{"-bad", "api"},
	}
	for _, c := range cases {
		t.Run(c.profile+"/"+c.tunnel, func(t *testing.T) {
			if _, err := UnitName(c.profile, c.tunnel); err == nil {
				t.Fatalf("UnitName(%q,%q) should fail", c.profile, c.tunnel)
			}
		})
	}
}

func TestUnitPath_StaysInsideSystemdDir(t *testing.T) {
	tmp := t.TempDir()
	withSystemdDir(t, tmp)

	name, err := UnitName("senvada", "api")
	if err != nil {
		t.Fatal(err)
	}
	got, err := UnitPath(name)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmp, "cfmux-senvada-api.service")
	if got != want {
		t.Fatalf("UnitPath = %q, want %q", got, want)
	}
}

func TestUnitPath_RejectsCraftedName(t *testing.T) {
	cases := []string{
		"",
		"cfmux-..-x.service",
		"cfmux-x/y.service",
		"cfmux-x.service\nfoo",
		"not-a-cfmux.service",
		"cfmux-x.notservice",
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			if _, err := UnitPath(c); err == nil {
				t.Fatalf("UnitPath(%q) should fail", c)
			}
		})
	}
}

func TestRenderUnit_GoldenSnapshot(t *testing.T) {
	got, err := RenderUnit(UnitOpts{
		Profile:     "senvada",
		Tunnel:      "api",
		User:        "alice",
		Group:       "users",
		CfmuxBinary: "/usr/local/bin/cfmux",
	})
	if err != nil {
		t.Fatalf("RenderUnit: %v", err)
	}

	// Required substrings — assert each independently so a failure tells you
	// exactly what's missing rather than a wall-of-diff.
	required := []string{
		"Description=Cfmux Tunnel api (profile senvada)",
		"User=alice",
		"Group=users",
		"ExecStart=/usr/local/bin/cfmux tunnel run api --profile senvada",
		"Restart=always",
		"NoNewPrivileges=true",
		"PrivateTmp=true",
		"ProtectSystem=full",
		"ProtectHome=no",
		"ProtectKernelTunables=true",
		"ProtectKernelModules=true",
		"ProtectControlGroups=true",
		"LockPersonality=true",
		"RestrictRealtime=true",
		"RestrictSUIDSGID=true",
		"WantedBy=multi-user.target",
		"# MemoryDenyWriteExecute intentionally omitted",
		"# ProtectSystem=full (not strict)",
		"# ProtectHome=no",
	}
	for _, s := range required {
		if !strings.Contains(got, s) {
			t.Errorf("rendered unit missing %q", s)
		}
	}

	// Must NOT contain the harder directives we deliberately downgraded.
	forbidden := []string{
		"ProtectSystem=strict",
		"ProtectHome=read-only",
		"MemoryDenyWriteExecute=",
	}
	for _, s := range forbidden {
		if strings.Contains(got, s) {
			t.Errorf("rendered unit unexpectedly contains %q", s)
		}
	}
}

func TestRenderUnit_RejectsBadInputs(t *testing.T) {
	base := UnitOpts{Profile: "p", Tunnel: "t", User: "alice", Group: "users", CfmuxBinary: "/x"}

	bad := []struct {
		name string
		mut  func(o *UnitOpts)
	}{
		{"bad-profile", func(o *UnitOpts) { o.Profile = "p;rm" }},
		{"bad-tunnel", func(o *UnitOpts) { o.Tunnel = "t\nbad" }},
		{"bad-user", func(o *UnitOpts) { o.User = "alice;ls" }},
		{"bad-group", func(o *UnitOpts) { o.Group = "g$x" }},
		{"relative-binary", func(o *UnitOpts) { o.CfmuxBinary = "cfmux" }},
		{"empty-binary", func(o *UnitOpts) { o.CfmuxBinary = "" }},
		{"binary-newline", func(o *UnitOpts) { o.CfmuxBinary = "/usr/bin/x\nfoo" }},
	}
	for _, c := range bad {
		t.Run(c.name, func(t *testing.T) {
			o := base
			c.mut(&o)
			if _, err := RenderUnit(o); err == nil {
				t.Fatalf("RenderUnit(%+v) should fail", o)
			}
		})
	}
}

func TestWriteUnit_RefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfmux-x-y.service")

	if err := WriteUnit(path, "first"); err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	if err := WriteUnit(path, "second"); err == nil {
		t.Fatal("second write should fail (O_EXCL)")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "first" {
		t.Fatalf("file was overwritten: %q", string(b))
	}
}

func TestWriteUnit_RequiresAbsolutePath(t *testing.T) {
	if err := WriteUnit("relative.service", "x"); err == nil {
		t.Fatal("WriteUnit should reject relative paths")
	}
}

func TestRemoveUnit_RefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.service")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := RemoveUnit(link); err == nil {
		t.Fatal("RemoveUnit should refuse symlinks")
	}
	// Symlink is still there.
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("symlink should not have been removed: %v", err)
	}
}

func TestRemoveUnit_RemovesRegularFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.service")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveUnit(path); err != nil {
		t.Fatalf("RemoveUnit on a regular file failed: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file should have been removed")
	}
}

// withSystemdDir redirects systemdDir for the duration of t and restores it.
func withSystemdDir(t *testing.T, dir string) {
	t.Helper()
	orig := systemdDir
	systemdDir = dir
	t.Cleanup(func() { systemdDir = orig })
}
