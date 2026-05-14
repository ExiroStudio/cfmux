# Cfmux

> A profile-aware wrapper and service manager for `cloudflared`

Cfmux is a small, opinionated CLI tool that sits in front of `cloudflared` to make multi-account Cloudflare Tunnel workflows manageable on Linux systems.

It does not replace `cloudflared`. It wraps it — handling profile isolation, credential routing, tunnel lifecycle, and systemd service generation so you do not have to do those things by hand.

---

## The Problem

`cloudflared` is a solid tool, but it is designed around a single account. When you run `cloudflared tunnel login`, it writes a `cert.pem` to `~/.cloudflared/`. If you manage multiple Cloudflare accounts — for different clients, projects, or environments — that single file becomes a constant source of confusion. There is no built-in concept of profiles, no way to isolate credentials per account, and no service generator that understands tunnel identity.

Cfmux fills that gap. It maintains isolated profiles on disk, automatically injects the right `--origincert` and `--config` flags when delegating to `cloudflared`, and generates per-tunnel systemd units from a consistent template.

If `cloudflared` eventually supports native multi-account profile management, Cfmux would happily step aside. Until then, it handles the coordination layer.

---

## Requirements

- Linux (x86\_64 or arm64)
- [`cloudflared`](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/) installed and on `$PATH`
- `systemd` (for service management features)

---

## Installation

Use the install script for both fresh installs and updates:

```bash
curl -fsSL https://raw.githubusercontent.com/ExiroStudio/cfmux/main/install.sh | bash
```

This downloads the latest release binary for your platform, installs it to `/usr/local/bin/cfmux`, and verifies the install by running `cfmux version`.

To confirm the installation:

```bash
cfmux version
```

---

## Quick Start

```bash
# 1. Create a profile and authenticate with your Cloudflare account
cfmux profile add personal

# 2. Switch to that profile
cfmux profile use personal

# 3. Create a tunnel under the active profile
cfmux tunnel create myapp

# 4. Edit the generated config to define your ingress rules
#    (file path is printed after creation)

# 5. Run the tunnel
cfmux tunnel run myapp

# 6. When ready, install and enable a systemd service for it
sudo cfmux service install myapp
sudo cfmux service enable myapp

# 7. Check service status
cfmux service status myapp
```

---

## How It Works

Cfmux stores all state under `~/.cfmux/`. Each profile is a self-contained directory with its own certificate, tunnel credentials, tunnel configs, and a local registry.

```text
~/.cfmux/
├── profiles/
│   ├── personal/
│   │   ├── cert.pem              # Origin certificate from cloudflared login
│   │   ├── tunnels/              # Tunnel credential files (one JSON per tunnel)
│   │   │   └── myapp.json
│   │   ├── configs/              # Tunnel config files (one YAML per tunnel)
│   │   │   └── myapp.yml
│   │   └── registry.json         # Local tunnel inventory for this profile
│   │
│   └── client-a/
│       └── ...
│
└── current-profile               # Active profile name (plain text)
```

When you run a tunnel command, Cfmux reads the active profile, resolves the correct `cert.pem` and config path from the registry, and passes everything to `cloudflared` with the right flags. You do not have to think about which certificate belongs to which account.

### Tunnel States

When you run `cfmux tunnel list`, Cfmux merges the local registry with the live Cloudflare API and classifies each tunnel:

| State | Meaning |
|---|---|
| `managed` | Exists in local registry and confirmed on Cloudflare — normal state |
| `unmanaged` | Exists on Cloudflare but not in local registry — created outside Cfmux |
| `stale` | Exists in local registry but no longer on Cloudflare — deleted externally |

Service installation is intentionally blocked for `stale` and `unmanaged` tunnels.

---

## Commands

### Profile Management

```bash
cfmux profile add <name>
```
Creates a new profile directory and runs `cloudflared tunnel login` interactively. The resulting certificate is imported into the profile's isolated directory.

```bash
cfmux profile list
```
Lists all profiles. The active profile is marked with `→`.

```bash
cfmux profile current
```
Prints the name of the currently active profile.

```bash
cfmux profile use <name>
```
Switches the active profile.

```bash
cfmux profile remove <name>
```
Deletes a profile and all its contents. Refuses to remove the currently active profile.

---

### Tunnel Management

All tunnel commands operate on the active profile unless otherwise noted.

```bash
cfmux tunnel create <name>
```
Creates a new tunnel via `cloudflared tunnel create`, stores credentials in the profile's `tunnels/` directory, and generates a starter config in `configs/`. The tunnel is registered in `registry.json`. You edit the generated config file to add your ingress rules.

```bash
cfmux tunnel list
```
Lists tunnels from the local registry merged with remote Cloudflare state. Shows name, UUID, and state.

```bash
cfmux tunnel run <name>
```
Runs the tunnel. Cfmux resolves the credentials and config from the registry and delegates to `cloudflared` with the correct flags. Accepts `--profile <name>` to override the active profile.

```bash
cfmux tunnel delete <name>
```
Deletes the tunnel from Cloudflare and removes local credentials and config. Also cleans up the registry entry.

**Unknown tunnel subcommands are passed through to `cloudflared` automatically.** For example:

```bash
cfmux tunnel info <uuid>
# equivalent to: cloudflared tunnel info <uuid>
```

---

### Service Management

Service commands require root (via `sudo`) except for `status`.

```bash
sudo cfmux service install <tunnel>
```
Generates and installs a systemd unit file at `/etc/systemd/system/cfmux-<profile>-<tunnel>.service`. The unit runs `cfmux tunnel run <tunnel> --profile <profile>` and applies several systemd hardening options. Does not start or enable the service — that is a separate step.

Before installing, Cfmux runs a series of preflight checks:

- Tunnel exists in registry and is confirmed on Cloudflare (`managed` state)
- Credentials and config files are present
- The `cfmux` binary is not in a temporary or home directory
- Binary ownership and permissions are acceptable

```bash
sudo cfmux service enable <tunnel>
```
Enables and starts the service (`systemctl enable --now`).

```bash
cfmux service status <tunnel>
```
Prints the systemctl status for the tunnel's unit. Does not require root. Exit codes are propagated from systemctl (0 = active, 3 = inactive, 4 = unit not found).

```bash
sudo cfmux service uninstall <tunnel>
```
Stops, disables, and removes the unit file. Does not touch tunnel credentials, configs, or the registry — only removes the service unit.

---

## Tunnel Config

When you create a tunnel, Cfmux generates a starter config at `~/.cfmux/profiles/<profile>/configs/<tunnel>.yml`:

```yaml
tunnel: <uuid>
credentials-file: /home/user/.cfmux/profiles/personal/tunnels/myapp.json

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
```

Edit this file to define your ingress rules before running the tunnel.

---

## Architecture Notes

Cfmux has no dependencies beyond [`github.com/spf13/cobra`](https://github.com/spf13/cobra) for CLI parsing. All interaction with Cloudflare happens through `cloudflared` subprocesses — Cfmux does not use the Cloudflare API directly, except to read tunnel state for the list and merge logic.

The binary is a single static Go executable with no runtime dependencies. Build targets are `linux/amd64` and `linux/arm64`.

---

## Building from Source

```bash
git clone https://github.com/ExiroStudio/cfmux.git
cd cfmux
go build -o cfmux .
```

Release builds inject version metadata at link time:

```bash
go build \
  -ldflags "-s -w \
    -X 'cfmux/internal/version.Version=v1.0.0' \
    -X 'cfmux/internal/version.Commit=abc1234' \
    -X 'cfmux/internal/version.Date=2025-01-01T00:00:00Z'" \
  -o cfmux .
```

---

## Disclaimer

Cfmux is an independent open-source project and is not affiliated with or endorsed by Cloudflare, Inc. `cloudflared` and Cloudflare Tunnel are products of Cloudflare, Inc.

---

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.
