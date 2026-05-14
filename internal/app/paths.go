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

func ProfileTunnelDir(profile string) string {
	return filepath.Join(ProfileDir(&profile), "tunnels")
}

func ProfileConfigDir(profile string) string {
	return filepath.Join(ProfileDir(&profile), "configs")
}

func TunnelCredsPath(profile, name string) string {
	return filepath.Join(ProfileTunnelDir(profile), name+".json")
}

func TunnelConfigPath(profile, name string) string {
	return filepath.Join(ProfileConfigDir(profile), name+".yml")
}

func TunnelRegistryPath(profile string) string {
	return filepath.Join(ProfileDir(&profile), "registry.json")
}
