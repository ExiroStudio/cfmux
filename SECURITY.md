# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| v0.1.2 | ✅ |
| v0.1.1 | ✅ |
| v0.1.0 | ✅ |

Only the latest release receives security fixes. Users are encouraged to always upgrade to the latest version.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

To report a security vulnerability, open a [GitHub Security Advisory](https://github.com/ExiroStudio/cfmux/security/advisories/new) on this repository. If you are unsure whether an issue is a security vulnerability, err on the side of reporting it privately.

When reporting, please include:

- A clear description of the vulnerability
- Steps to reproduce the issue
- Potential impact and attack scenarios
- Any suggested mitigations or patches (optional)

You can expect an initial response within **5 business days** and a resolution or status update within **30 days**.

## Security Design

> cfmux is designed for trusted-host environments and does not attempt to defend against attackers with full root access.

### Trust Boundaries

cfmux is a **CLI tool** with no network listeners. All network operations are delegated to the `cloudflared` subprocess. The attack surface is limited to:

- Local filesystem access (profile directories under `~/.cfmux/`)
- `cloudflared` subprocess execution
- systemd unit file management (requires root)

### Credential Isolation

Each Cloudflare account profile is stored in an isolated directory (`~/.cfmux/profiles/<profile>/`). This means:

- Credentials (`cert.pem`) and tunnel JSON secrets are never shared across profiles.
- After a `cfmux profile add`, the temporary `~/.cloudflared/cert.pem` is moved (not copied) into the profile directory to prevent stale credential exposure.
- Tunnel credential files are written with permission `0o600` (owner-read-only).

### Privilege Handling

- Service installation and management require root (`sudo`).
- cfmux uses `SUDO_USER` to correctly resolve the real user's home directory when invoked with `sudo`, preventing accidental operations in `/root/.cfmux/`.
- If a service would run as root, cfmux emits an explicit warning to stderr. Running services as root is supported for system-level deployments, but non-root runtimes are recommended where practical.

### Runtime Ownership Model

cfmux supports two ownership models:

- **User-owned deployments** — profiles and credentials live under `~/.cfmux/` and are managed without root.
- **Root-owned system deployments** — profiles live under `/root/.cfmux/` and services run as a system-level process.

When invoked through `sudo`, cfmux attempts to preserve the invoking user's ownership model using `SUDO_USER`, so that profile data is written to the real user's home directory rather than `/root`.

### systemd Service Hardening

Generated unit files apply the following hardening directives:

| Directive | Value | Reason |
|---|---|---|
| `NoNewPrivileges` | `true` | Prevents privilege escalation via setuid |
| `PrivateTmp` | `true` | Isolates `/tmp` from other services |
| `ProtectSystem` | `full` | Makes `/usr`, `/boot`, `/etc` read-only |
| `ProtectKernelTunables` | `true` | Blocks `/proc/sys` writes |
| `ProtectKernelModules` | `true` | Blocks kernel module loading |
| `ProtectControlGroups` | `true` | Restricts cgroup manipulation |
| `LockPersonality` | `true` | Prevents `personality(2)` syscall |
| `RestrictRealtime` | `true` | Prevents real-time scheduling |
| `RestrictSUIDSGID` | `true` | Blocks setuid/setgid bit creation |

**Known trade-offs:**

- `ProtectHome=no` — The service must read `~/.cfmux/` profile directories at runtime. A tighter `ReadOnlyPaths=` restriction is a planned improvement.
- `MemoryDenyWriteExecute` is intentionally omitted — the Go runtime and cloudflared may allocate executable memory pages for runtime internals or third-party components.
- `ProtectSystem=full` (not `strict`) — Ensures runtime compatibility with paths that `cloudflared` may need to access.

### Input Validation & Injection Prevention

cfmux validates all user-supplied names before writing to the filesystem or generating systemd unit files.

- Profile and tunnel names are restricted to `[A-Za-z0-9_-]`, max 64 characters.
- Path traversal sequences (`../`), shell metacharacters (`;`, `$()`, backticks), control characters, non-ASCII and potentially confusable Unicode characters are rejected.
- Generated unit file paths are verified to remain within `/etc/systemd/system/` after canonicalization.
- Unit files are created with `O_EXCL` (fails if the file already exists) to prevent overwrite races.
- Symlinks are rejected when removing unit files (guards against TOCTOU attacks).

### Service Binary Validation

Before installing a service, cfmux validates the target binary:

- Resolves symlinks to the real binary path.
- Rejects world-writable binaries (`0o002`).
- Rejects binaries located in world-writable or ephemeral directories (`/tmp`, `/var/tmp`, `/dev/shm`).
- Rejects binaries located inside user home directories.
- Validates binary ownership.

## Known Limitations

| Area | Limitation | Mitigation |
|---|---|---|
| `cert.pem` permissions | Written as `0o644` by `cloudflared`; readable by all local users | On multi-user systems, local users may read `cert.pem` files generated by `cloudflared` unless permissions are manually tightened after creation |
| Config YAML permissions | Tunnel config files are `0o644`; they do not contain secrets but do expose tunnel topology | Acceptable in single-user environments; restrict in shared environments |
| No audit logging | Tunnel creation, deletion, and profile switches are not logged to syslog/journald | Use shell history or `cloudflared` logs for audit trails |
| `ProtectHome=no` | The service user's home directory is accessible (read/write) to the process | A future release will add `ReadOnlyPaths=` to restrict writes |

## Dependencies

cfmux has a minimal dependency footprint:

- [`github.com/spf13/cobra`](https://github.com/spf13/cobra) — CLI framework
- [`cloudflared`](https://github.com/cloudflare/cloudflared) — external binary (not vendored; must be installed separately)

cfmux does **not** make direct Cloudflare API calls. All Cloudflare interactions are delegated to the `cloudflared` subprocess.

cfmux does **not** invoke shell interpreters (`sh -c`, `bash -c`) when executing `cloudflared` or `systemctl` commands. All subprocesses are executed using direct argv invocation, eliminating shell injection as an attack vector.

## Secure Installation

- Download releases only from the [official GitHub Releases page](https://github.com/ExiroStudio/cfmux/releases).
- Verify the binary is installed to a trusted, non-world-writable path (e.g., `/usr/local/bin/`).
- Do not run `cfmux` as root for day-to-day tunnel operations. Use `sudo cfmux service install` only when managing system services.
- Keep `cloudflared` updated independently, as cfmux relies on it for all tunnel operations.
