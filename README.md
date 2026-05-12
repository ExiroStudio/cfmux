# Cfmux

> Multi-profile wrapper and service manager for Cloudflare Tunnel (`cloudflared`)

Cfmux is a lightweight CLI tool designed to simplify multi-account and multi-profile workflows for `cloudflared`.

Instead of constantly switching certificates, configurations, or manually managing services, Cfmux provides isolated profiles and a cleaner workflow for managing Cloudflare Tunnels on Linux systems.

---

## Why Cfmux?

Managing multiple Cloudflare accounts with `cloudflared` can become messy very quickly.

Common problems:

* Overwriting `cert.pem`
* Switching accounts manually
* Confusing tunnel ownership
* Service management chaos
* Hard to automate multi-project setups

Cfmux solves this by introducing:

* Isolated profiles
* Profile switching
* Tunnel command passthrough
* Service generation
* Cleaner CLI workflow

---

# Features (Planned)

## Profile Management

Create and isolate multiple Cloudflare profiles.

```bash
cfmux profile add personal
cfmux profile add client-a
cfmux profile use personal
```

---

## Tunnel Command Wrapper

Run `cloudflared` commands automatically using the active profile.

```bash
cfmux tunnel list
cfmux tunnel run mytunnel
```

Internally:

```bash
cloudflared --origincert ~/.cfmux/profiles/personal/cert.pem tunnel list
```

---

## Service Generator

Generate and manage system services for tunnels.

```bash
cfmux service install personal
```

Example output:

```text
/etc/systemd/system/cfmux-personal.service
```

---

## Multi-Account Workflow

Designed for users managing:

* Multiple Cloudflare accounts
* Multiple clients
* Self-hosted infrastructure
* Homelabs
* VPS fleets

---

# Philosophy

Cfmux focuses on:

* Minimalism
* Unix-style workflow
* Single binary deployment
* Automation-friendly CLI
* Linux-first infrastructure tooling

No dashboard.
No unnecessary abstraction.
Just a clean developer experience.

---

# Project Status

> Early development / prototype stage

The project is currently focused on building the core architecture and CLI workflow.

Current priorities:

* Profile isolation
* Command passthrough
* Service generation
* Stable internal structure

---

# Planned Architecture

```text
cfmux/
├── cmd/
├── internal/
│   ├── profile/
│   ├── cloudflared/
│   ├── service/
│   └── config/
├── main.go
└── go.mod
```

---

# Storage Structure

```text
~/.cfmux/
├── profiles/
│   ├── personal/
│   │   ├── cert.pem
│   │   ├── config.yml
│   │   └── metadata.json
│   │
│   └── client-a/
│
└── current-profile
```

---

# Installation

> Coming soon

---

# Example Workflow

```bash
cfmux profile add personal

cfmux profile use personal

cfmux tunnel list

cfmux service install personal
```

---

# Goals

Cfmux aims to become:

* A practical daily-driver tool for Cloudflare Tunnel users
* A cleaner multi-account experience for `cloudflared`
* A lightweight infrastructure utility for Linux environments

---

# Disclaimer

Cfmux is an independent open-source project and is not affiliated with or endorsed by Cloudflare.

`cloudflared` and Cloudflare Tunnel are trademarks of Cloudflare, Inc.

---

# License

Apache License 2.0 (planned)
