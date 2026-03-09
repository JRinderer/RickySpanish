# RickySpanish

A local, single-user CLI project management tool written in Go. All data is
**encrypted at rest** using AES-256-GCM. No third-party libraries are used —
only the Go standard library.

## Features

- Track projects with name, priority, company-goal flag, status, notes, and a
  project directory
- Auto-generates project names from WWII and Korean War military operations when
  no name is provided
- Fully encrypted storage (AES-256-GCM)
- Encryption key stored in the **system keychain** — never in source code or a
  plaintext file
- **MCP server mode** — expose projects as Claude tools so you can manage them
  conversationally via Claude
- Cross-platform: macOS, Linux, Windows

---

## Quick Start

```bash
# Build
go build -o rickspanish .

# Add a project (name auto-generated)
./rickspanish add

# Add a project with options
./rickspanish add --name "Website Redesign" --priority high --company-goal --dir ~/projects/website

# List projects
./rickspanish list

# Filter
./rickspanish list --status active --priority high

# View a project
./rickspanish get <id-or-name>

# Update a project
./rickspanish update <id> --status completed

# Add a note
./rickspanish note <id> "Kickoff meeting went well"

# List notes
./rickspanish notes <id>

# Delete a note
./rickspanish delete-note <project-id> <note-id>

# Delete a project
./rickspanish delete <id-or-name>
```

---

## Encryption & Key Management

RickySpanish uses **AES-256-GCM** to encrypt the entire project database before
writing it to disk. The encryption key is **64-character hex-encoded random
bytes (32 bytes)** and is **never** stored in source code or a plaintext file.

### macOS

The key is stored in the **macOS Keychain** automatically on first run.

- Service name: `rickspanish`
- Account name: `encryption-key`

You can view or manage it in **Keychain Access.app** or via the terminal:

```bash
# View the stored key
security find-generic-password -s rickspanish -a encryption-key -w

# Delete the key (e.g., to rotate it)
security delete-generic-password -s rickspanish -a encryption-key
```

> **Warning:** Deleting the key makes existing data unrecoverable unless you
> back up the key first.

### Linux

RickySpanish uses **GNOME Keyring via `secret-tool`** on Linux.

**Install the required tool:**

```bash
# Debian / Ubuntu
sudo apt install libsecret-tools

# Fedora / RHEL
sudo dnf install libsecret

# Arch Linux
sudo pacman -S libsecret
```

On first run the key is generated and stored in GNOME Keyring automatically.

**Fallback — environment variable:**

If `secret-tool` is not available (e.g., on a headless server), set the
`RICKSPANISH_ENCRYPTION_KEY` environment variable to a 64-character hex string:

```bash
# Generate a key
openssl rand -hex 32

# Set it (add to ~/.bashrc or ~/.zshrc or a secrets manager)
export RICKSPANISH_ENCRYPTION_KEY=<your-64-char-hex-key>
```

> Store this key securely (e.g., in a password manager). Losing it means
> losing access to your data.

### Windows

The key is stored in **Windows Credential Manager** via PowerShell on first run.

- Target: `rickspanish/encryption-key`

You can view it in **Control Panel → Credential Manager → Windows Credentials**.

**Fallback — environment variable:**

If PowerShell credential storage fails, set:

```powershell
$env:RICKSPANISH_ENCRYPTION_KEY = "<your-64-char-hex-key>"
# Or permanently via System Properties → Environment Variables
```

---

## Data Location

| Platform | Path |
|----------|------|
| macOS / Linux | `~/.local/share/rickspanish/projects.enc` |
| Linux (XDG) | `$XDG_DATA_HOME/rickspanish/projects.enc` |
| Windows | `%APPDATA%\rickspanish\projects.enc` |

---

## MCP Server (Claude Integration)

RickySpanish can act as an **MCP (Model Context Protocol) server**, letting you
manage projects conversationally through Claude.

### Start the server

```bash
./rickspanish serve
```

The server communicates over **stdio** using JSON-RPC 2.0, which is the
standard MCP transport.

### Configure Claude Desktop

Add the following to your Claude Desktop configuration file:

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
**Linux:** `~/.config/Claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "rickspanish": {
      "command": "/absolute/path/to/rickspanish",
      "args": ["serve"]
    }
  }
}
```

### Available MCP Tools

| Tool | Description |
|------|-------------|
| `add_project` | Create a new project |
| `list_projects` | List projects with optional filters |
| `get_project` | Get full details of a project |
| `update_project` | Update project fields |
| `delete_project` | Delete a project |
| `add_note` | Add a note to a project |
| `delete_note` | Delete a note from a project |

### Example Claude conversation

> **You:** Add a new high-priority project called "Q2 Product Launch" that's
> related to our company goals.
>
> **Claude:** *(calls `add_project`)* Created project "Q2 Product Launch" with
> high priority, marked as a company goal. ID: `a1b2c3d4...`
>
> **You:** What are my active projects?
>
> **Claude:** *(calls `list_projects` with status=active)* You have 3 active
> projects: ...

---

## Priority Values

| Value | Meaning |
|-------|---------|
| `low` | Nice to have |
| `medium` | Normal work item (default) |
| `high` | Urgent or critical |

## Status Values

| Value | Meaning |
|-------|---------|
| `active` | Currently being worked on (default) |
| `on_hold` | Paused |
| `completed` | Done |
| `archived` | Archived, no longer relevant |

---

## Building from Source

Requires **Go 1.21+**. No external dependencies.

```bash
git clone <repo>
cd RickySpanish
go build -o rickspanish .
```

Cross-compile examples:

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o rickspanish-linux .

# Windows
GOOS=windows GOARCH=amd64 go build -o rickspanish.exe .

# macOS ARM (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o rickspanish-arm64 .
```

---

## Backup & Recovery

To back up your data:

1. Copy `projects.enc` to a safe location
2. Back up your encryption key from the system keychain

To restore:

1. Place `projects.enc` back in the data directory
2. Ensure the same encryption key is available in your keychain (or env var)

---

## Security Notes

- The database file is AES-256-GCM encrypted — even if the file is copied,
  it cannot be read without the key
- Each write generates a new random 96-bit nonce, preventing replay attacks
- The encryption key is generated using `crypto/rand` (cryptographically secure)
- File writes are atomic (write to `.tmp`, then rename) to prevent corruption
