package profile

import (
	"cfmux/internal/app"
	"os"
)

func List() ([]string, error) {
	entries, err := os.ReadDir(app.ProfileDir(nil))
	if err != nil {
		return nil, err
	}

	var profiles []string

	for _, entry := range entries {
		if entry.IsDir() {
			profiles = append(profiles, entry.Name())
		}
	}

	return profiles, nil
}
