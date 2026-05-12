package profile

import (
	"cfmux/internal/app"
	"os"
	"strings"
)

func Current() (string, error) {
	data, err := os.ReadFile(app.CurrentProfileFile())
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}
