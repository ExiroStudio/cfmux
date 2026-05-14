package app

import (
	"os"
	"os/user"
)

// ResolveHome returns the home directory of the current user, or the home
// directory of the user who invoked sudo if running as root.
func ResolveHome() (string, error) {
	if os.Geteuid() == 0 {
		sudoUser := os.Getenv("SUDO_USER")

		if sudoUser != "" {
			u, err := user.Lookup(sudoUser)
			if err != nil {
				return "", err
			}

			return u.HomeDir, nil
		}
	}
	return os.UserHomeDir()
}
