package tunnel

import (
	"cfmux/internal/app"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func Delete(profile, name string, progress app.ProgressFunc) error {
	reg, err := Load(profile)
	if err != nil {
		return err
	}
	t, ok := reg.Find(name)
	if !ok {
		return fmt.Errorf("tunnel %q not registered", name)
	}

	cert := filepath.Join(app.ProfileDir(&profile), "cert.pem")

	progress("Deleting tunnel via cloudflared")
	cmd := exec.Command("cloudflared",
		"--origincert", cert,
		"tunnel", "delete", t.UUID,
	)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := cmd.Run(); err != nil {
		return err
	}

	progress("Removing local files")
	_ = os.Remove(t.AbsCreds(profile))
	_ = os.Remove(t.AbsConfig(profile))

	progress("Updating registry")
	_ = reg.Remove(name)
	return Save(profile, reg)
}
