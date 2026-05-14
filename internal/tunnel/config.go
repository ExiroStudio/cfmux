package tunnel

import (
	"fmt"
	"os"
)

const configTemplate = `tunnel: %s
credentials-file: %s

ingress:
  # Add ingress rules above the catch-all. Examples:
  #
  # - hostname: app.example.com
  #   service: http://localhost:3000
  #
  # - hostname: api.example.com
  #   path: /v1/*
  #   service: http://localhost:8080
  #
  # - hostname: ssh.example.com
  #   service: ssh://localhost:22
  #
  # Catch-all (must be last):
  - service: http_status:404
`

func writeDefaultConfig(path, uuid, credsPath string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, configTemplate, uuid, credsPath)
	return err
}
