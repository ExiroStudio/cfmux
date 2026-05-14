package tunnel

import (
	"cfmux/internal/app"
	"path/filepath"
)

const CurrentSchemaVersion = 1

type Tunnel struct {
	Name       string `json:"name"`
	RemoteName string `json:"remote_name"`
	UUID       string `json:"uuid"`
	Config     string `json:"config"`
	Creds      string `json:"creds"`
}

func (t Tunnel) AbsConfig(profile string) string {
	return filepath.Join(app.ProfileDir(&profile), t.Config)
}

func (t Tunnel) AbsCreds(profile string) string {
	return filepath.Join(app.ProfileDir(&profile), t.Creds)
}

type Registry struct {
	SchemaVersion int      `json:"schema_version"`
	Tunnels       []Tunnel `json:"tunnels"`
}
