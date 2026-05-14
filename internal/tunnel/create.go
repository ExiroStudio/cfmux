package tunnel

import (
	"cfmux/internal/app"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Create(profile, name string, progress app.ProgressFunc) error {
	reg, err := Load(profile)
	if err != nil {
		return err
	}
	if _, exists := reg.Find(name); exists {
		return fmt.Errorf("tunnel %q already registered", name)
	}

	if err := os.MkdirAll(app.ProfileTunnelDir(profile), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(app.ProfileConfigDir(profile), 0755); err != nil {
		return err
	}

	credsPath := app.TunnelCredsPath(profile, name)
	configPath := app.TunnelConfigPath(profile, name)
	cert := filepath.Join(app.ProfileDir(&profile), "cert.pem")

	if _, err := os.Stat(credsPath); err == nil {
		return fmt.Errorf("credentials already exist at %s; remove manually or pick a different name", credsPath)
	}
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config already exists at %s; remove manually or pick a different name", configPath)
	}

	cfHome, err := defaultCloudflaredDir()
	if err != nil {
		return err
	}
	before := snapshotJSON(cfHome)

	progress("Creating tunnel via cloudflared")
	cmd := exec.Command("cloudflared",
		"--origincert", cert,
		"tunnel", "create",
		"--credentials-file", credsPath,
		name,
	)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := cmd.Run(); err != nil {
		return err
	}

	progress("Verifying credentials location")
	if _, err := os.Stat(credsPath); err != nil {
		if mErr := migrateStrayCreds(cfHome, before, credsPath); mErr != nil {
			return fmt.Errorf(
				"credentials not at %s after create; cloudflared may have ignored --credentials-file: %w",
				credsPath, mErr,
			)
		}
		progress("Migrated stray credentials from ~/.cloudflared")
	}

	progress("Reading tunnel UUID")
	uuid, err := readUUIDFromCreds(credsPath)
	if err != nil {
		return err
	}

	progress("Generating tunnel config")
	if err := writeDefaultConfig(configPath, uuid, credsPath); err != nil {
		return err
	}

	progress("Updating registry")
	if err := reg.Add(Tunnel{
		Name:       name,
		RemoteName: name,
		UUID:       uuid,
		Config:     filepath.Join("configs", name+".yml"),
		Creds:      filepath.Join("tunnels", name+".json"),
	}); err != nil {
		return err
	}
	return Save(profile, reg)
}

func defaultCloudflaredDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cloudflared"), nil
}

func snapshotJSON(dir string) map[string]struct{} {
	out := map[string]struct{}{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		out[e.Name()] = struct{}{}
	}
	return out
}

func migrateStrayCreds(cfHome string, before map[string]struct{}, dest string) error {
	entries, err := os.ReadDir(cfHome)
	if err != nil {
		return err
	}

	var newFiles []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if _, existed := before[e.Name()]; existed {
			continue
		}
		newFiles = append(newFiles, e.Name())
	}

	if len(newFiles) == 0 {
		return errors.New("no new credentials file found in ~/.cloudflared")
	}
	if len(newFiles) > 1 {
		return fmt.Errorf("multiple new files in ~/.cloudflared, cannot disambiguate: %v", newFiles)
	}

	return os.Rename(filepath.Join(cfHome, newFiles[0]), dest)
}

type credsFile struct {
	TunnelID string `json:"TunnelID"`
}

func readUUIDFromCreds(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var c credsFile
	if err := json.Unmarshal(data, &c); err != nil {
		return "", fmt.Errorf("parse credentials %s: %w", path, err)
	}
	if c.TunnelID == "" {
		return "", fmt.Errorf("credentials %s missing TunnelID", path)
	}
	return c.TunnelID, nil
}
