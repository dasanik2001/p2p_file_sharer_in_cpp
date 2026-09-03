# Transfera CLI (Go)

A command-line client for [Transfera](../README.md) — secure P2P file sharing, optimized for large file transfers (100MB+) with minimal memory usage.

## Quick Start

### Prerequisites

- [Go 1.21+](https://go.dev/dl/) installed
- Connects out-of-the-box to the official production server (`https://transfera-api.onrender.com`). No local server setup required!

### Install & Build

```bash
cd CLI_go

# Download dependencies
go mod tidy

# Build the binary
go build -o transfera.exe .     # Windows
go build -o transfera .         # Linux/macOS
```

### Usage (Cross-Device Transfer, e.g. Laptop to Desktop)

1. **On your Laptop** (Upload a photo):
   ```bash
   ./transfera upload my_photo.jpg
   ```
   Output:
   ```
   ✓ File ready to share!
   ┌────────────────────────────────────────
   │  Invite code: 55228
   │  Max downloads: 1
   └────────────────────────────────────────
   ```

2. **On your Desktop** (Download the photo):
   ```bash
   ./transfera download 55228
   # Or specify an output directory
   ./transfera download 55228 -o ~/Pictures/
   ```

3. **Check Server Health**:
   ```bash
   ./transfera health
   ```

## Commands

### `transfera health`

Check if the API server is reachable.

```bash
transfera health
transfera --api http://127.0.0.1:8080 health  # to check a local dev server
```

### `transfera upload <file>`

Upload a file (photo, video, archive, document) and receive an invite code to share.

```bash
transfera upload vacation.jpg                  # Basic photo upload
transfera upload video.mp4 -n 5               # Allow 5 downloads
transfera upload huge.zip --max-size 500       # Allow files up to 500MB
```

**Flags:**
| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--max-downloads` | `-n` | 1 | Maximum downloads allowed (1-100) |
| `--max-size` | `-s` | 100 | Max file size in MB |

### `transfera download <invite-code>`

Download a file using an invite code shared with you.

```bash
transfera download 52341                       # Download to current directory
transfera download 52341 -o ./received/        # Download to specific directory (creates it if missing)
transfera download 52341 --output-name doc.pdf # Override filename
```

**Flags:**
| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `.` | Output directory (automatically created if it doesn't exist) |
| `--output-name` | | | Override download filename |

## Global Flags & Environment Variables

Available on all commands:

| Flag | Short | Default / Fallback | Description |
|------|-------|--------------------|-------------|
| `--api` | `-a` | `https://transfera-api.onrender.com` | API server URL (can also be set via `TRANSFERA_API_URL` or `NEXT_PUBLIC_API_BASE_URL`) |
| `--verbose` | `-V` | `false` | Enable verbose output (shows HTTP headers, timing) |

## Cross-Compilation

Build for any platform from any platform:

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o transfera-linux .

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o transfera-macos .

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o transfera-macos-arm .

# Windows
GOOS=windows GOARCH=amd64 go build -o transfera.exe .
```

## Architecture

```
CLI_go/
├── main.go                      # Entry point (calls cmd.Execute())
├── cmd/
│   ├── root.go                  # Root command, global flags
│   ├── health.go                # Health check command
│   ├── upload.go                # Upload command
│   └── download.go              # Download command
├── internal/
│   ├── api/
│   │   └── client.go            # HTTP client (upload, download, health)
│   ├── progress/
│   │   └── bar.go               # Terminal progress bars
│   └── validation/
│       └── file.go              # File validation (size, existence)
├── go.mod                       # Go module definition
└── README.md                    # This file
```

## Memory Usage

The CLI uses streaming (io.Pipe) for uploads and downloads. Memory usage stays flat regardless of file size:

| File Size | CLI Memory | Browser Memory |
|-----------|-----------|----------------|
| 10 MB     | ~10 MB    | ~40 MB         |
| 100 MB    | ~10 MB    | ~400 MB        |
| 500 MB    | ~10 MB    | ❌ may crash    |
