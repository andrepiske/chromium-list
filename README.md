# chromium-list

NOTE: I'm not a Go/Golang dev. This code is mostly vibe-coded with some guidance of mine.

------

A lightweight CLI tool written in Go to find and retrieve the direct download URL for the latest Chromium browser .deb package from the Debian registry.

## Features

- **Automated Latest Version Check**: Fetches and parses the Debian package pool to find the absolute latest version of Chromium.
- **Architecture Detection**: Automatically detects your system architecture (amd64, arm64, etc.).
- **Cross-Platform Support**: Manually specify target architectures via flags.
- **Zero Runtime Dependencies**: Compiles to a single binary.

## Installation

### Prerequisites
- Go 1.21 or later

### Build from Source

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/chromium-list.git
   cd chromium-list
   ```

2. Build the binary:
   ```bash
   go mod tidy
   go build -o chromium-list
   ```

## Usage

Run the tool to get the download URL for the latest Chromium package for your current architecture:

```bash
./chromium-list
```

### Options

| Flag | Description | Default |
|------|-------------|---------|
| `--arch` | Target architecture (`amd64`, `arm64`, `armhf`, `i386`) | **Detected** (Host Arch) |
| `--help` | Show help message | N/A |

### Examples

**Get URL for current machine:**
```bash
./chromium-list
# Output: https://deb.debian.org/debian/pool/main/c/chromium/chromium_133.0.6943.53-1~deb12u1_amd64.deb
```

**Get URL for ARM64 (e.g., Raspberry Pi):**
```bash
./chromium-list --arch=arm64
```

**Download the latest package directly:**
```bash
wget $(./chromium-list)
```

## CI/CD

This project includes a GitHub Action to automatically build and test releases for multiple architectures (`amd64`, `arm64`).

