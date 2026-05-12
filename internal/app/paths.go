package app

import (
	"os"
	"path/filepath"
)

func BaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return filepath.Join(home, ".cfmux")
}

func ProfileDir(name *string) string {
	if name == nil {
		return filepath.Join(BaseDir(), "profiles")
	}
	return filepath.Join(BaseDir(), "profiles", *name)
}

func CurrentProfileFile() string {
	return filepath.Join(BaseDir(), "current-profile")
}
