package profile

import (
	"cfmux/internal/app"
	"errors"
	"os"
)

func Use(name string) error {
	if _, err := os.Stat(app.ProfileDir(&name)); err != nil {
		return errors.New("profile not found")
	}

	return os.WriteFile(
		app.CurrentProfileFile(),
		[]byte(name),
		0644,
	)
}
