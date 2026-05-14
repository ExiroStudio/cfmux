package tunnel

import (
	"bytes"
	"cfmux/internal/app"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
)

type RemoteTunnel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func fetchRemote(profile string) ([]RemoteTunnel, error) {
	cert := filepath.Join(app.ProfileDir(&profile), "cert.pem")

	var stdout bytes.Buffer
	cmd := exec.Command("cloudflared",
		"--origincert", cert,
		"tunnel", "list", "--output", "json",
	)
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cloudflared tunnel list: %w", err)
	}

	var remote []RemoteTunnel
	if err := json.Unmarshal(stdout.Bytes(), &remote); err != nil {
		return nil, fmt.Errorf("parse cloudflared output: %w", err)
	}

	return remote, nil
}
