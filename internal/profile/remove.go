package profile

import (
	"cfmux/internal/app"
	"errors"
	"os"
)

func Remove(name string) error {
	if _, err := os.Stat(app.ProfileDir(&name)); err != nil {
		return errors.New("profile not found")
	}

	current, err := Current()
	if err == nil && current == name {
		return errors.New("cannot remove active profile")
	}

	return os.RemoveAll(app.ProfileDir(&name))
}
