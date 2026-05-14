package tunnel

import (
	"cfmux/internal/app"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func Run(profile, name string) error {
	reg, err := Load(profile)
	if err != nil {
		return err
	}
	t, ok := reg.Find(name)
	if !ok {
		return fmt.Errorf("tunnel %q not registered", name)
	}

	credsAbs := t.AbsCreds(profile)
	configAbs := t.AbsConfig(profile)
	if _, err := os.Stat(credsAbs); err != nil {
		return fmt.Errorf("tunnel credentials missing at %s: %w", credsAbs, err)
	}
	if _, err := os.Stat(configAbs); err != nil {
		return fmt.Errorf("tunnel config missing at %s: %w", configAbs, err)
	}

	cert := filepath.Join(app.ProfileDir(&profile), "cert.pem")
	cmd := exec.Command("cloudflared",
		"--origincert", cert,
		"--config", configAbs,
		"tunnel", "run", t.UUID,
	)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	return cmd.Run()
}
