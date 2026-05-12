package cloudflared

import (
	"cfmux/internal/app"
	"cfmux/internal/profile"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func Execute(args ...string) error {
	current, err := profile.Current()
	if err != nil {
		return err
	}

	profileDir := app.ProfileDir(&current)

	cert := filepath.Join(profileDir, "cert.pem")
	config := filepath.Join(profileDir, "config.yml")

	finalArgs := []string{
		"--origincert", cert,
	}

	if _, err := os.Stat(config); err == nil {
		finalArgs = append(finalArgs,
			"--config", config,
		)
	}

	finalArgs = append(finalArgs, args...)
	fmt.Println("Running: cloudflared", finalArgs)
	cmd := exec.Command("cloudflared", finalArgs...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}
