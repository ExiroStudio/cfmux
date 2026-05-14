package profile

import (
	"cfmux/internal/app"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

func Add(name string, progress app.ProgressFunc) error {
	if _, err := os.Stat(app.ProfileDir(&name)); err == nil {
		return errors.New("profile already exists")
	}

	progress("Cleaning up temporary certificate")
	if err := cleanupTempCertificate(); err != nil {
		return err
	}

	progress("Creating profile directory")
	if err := createProfileDir(name); err != nil {
		return err
	}

	progress("Logging in to Cloudflare")
	if err := loginCloudflare(); err != nil {
		return err
	}

	progress("Importing certificate")
	if err := importCertificate(name); err != nil {
		return err
	}

	progress("Verifying certificate")
	if err := verifyCertificate(name); err != nil {
		return err
	}

	progress("Cleaning up temporary certificate")
	if err := cleanupTempCertificate(); err != nil {
		return err
	}

	return nil
}

func createProfileDir(name string) error {
	return os.MkdirAll(app.ProfileDir(&name), 0755)
}

func loginCloudflare() error {
	cmd := exec.Command("cloudflared", "tunnel", "login")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func importCertificate(name string) error {
	home, err := app.ResolveHome()
	if err != nil {
		return err
	}

	source := filepath.Join(home, ".cloudflared", "cert.pem")
	dest := filepath.Join(app.ProfileDir(&name), "cert.pem")

	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}

	return os.WriteFile(dest, data, 0644)
}

func verifyCertificate(name string) error {
	_, err := os.Stat(filepath.Join(app.ProfileDir(&name), "cert.pem"))
	return err
}

func cleanupTempCertificate() error {
	home, err := app.ResolveHome()
	if err != nil {
		return err
	}

	err = os.Remove(filepath.Join(home, ".cloudflared", "cert.pem"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
