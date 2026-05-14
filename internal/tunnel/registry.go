package tunnel

import (
	"cfmux/internal/app"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

func Load(profile string) (*Registry, error) {
	path := app.TunnelRegistryPath(profile)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Registry{SchemaVersion: CurrentSchemaVersion}, nil
		}
		return nil, err
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if reg.SchemaVersion == 0 {
		reg.SchemaVersion = CurrentSchemaVersion
	}

	return &reg, nil
}

func Save(profile string, reg *Registry) error {
	if err := os.MkdirAll(app.ProfileDir(&profile), 0755); err != nil {
		return err
	}

	reg.SchemaVersion = CurrentSchemaVersion

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(app.TunnelRegistryPath(profile), data, 0644)
}

func (r *Registry) Find(name string) (*Tunnel, bool) {
	for i := range r.Tunnels {
		if r.Tunnels[i].Name == name {
			return &r.Tunnels[i], true
		}
	}
	return nil, false
}

func (r *Registry) Add(t Tunnel) error {
	if _, exists := r.Find(t.Name); exists {
		return fmt.Errorf("tunnel %q already registered", t.Name)
	}
	r.Tunnels = append(r.Tunnels, t)
	return nil
}

func (r *Registry) Remove(name string) error {
	for i := range r.Tunnels {
		if r.Tunnels[i].Name == name {
			r.Tunnels = append(r.Tunnels[:i], r.Tunnels[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("tunnel %q not registered", name)
}
