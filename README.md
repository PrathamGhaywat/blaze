# Blaze

A cross-platform(untested on macOS) package manager for installing and managing command-line tools.

## Overview

Blaze downloads, verifies, and manages packages across Windows and  Linux. Packages are defined via JSON manifests that specify download URLs, SHA256 checksums, and binary locations for each platform.

Key features:
- SHA256 verification for all downloads
- Atomic installs with rollback on failure
- Multiple version support per package
- Easy version switching
- Automatic cleanup of stale entries

## Installation

Download the appropriate binary for your system:
- Windows: `blaze.exe`
- Linux: `blaze`

Place it somewhere in your PATH or run it directly.

## Quick Start

### Add a package
To test it out you can run the following command to install the GitHub CLI tool: 
```bash
blaze add https://raw.githubusercontent.com/PrathamGhaywat/blaze/refs/heads/main/test/test-manifest.json
```

Blaze will:
1. Download the manifest (in our case the github cli manifest from the test directory)
2. Select the appropriate platform target
3. Download and verify the package (SHA256)
4. Extract the archive
5. Add binaries to PATH
6. Register in the local registry

### List installed packages

```bash
blaze list
```

Shows all installed packages and versions.

### Switch between versions

```bash
blaze use package@version
```

Removes the current version from PATH and activates the specified version. A new shell is required for changes to take effect.

### Remove a package

```bash
blaze remove package-name
```

For a specific version:
```bash
blaze remove <package> [version]
```

To remove all versions:
```bash
blaze remove package-name --all
```

### Cleanup

```bash
blaze cleanup
```

Removes stale registry entries, dead PATH entries, and empty directories.

## Manifest Format

Manifests define packages and their platform-specific implementations.

```json
{
  "schema": 1,
  "name": "gh",
  "version": "2.87.3",
  "description": "GitHub CLI",
  "homepage": "https://github.com/cli/cli",
  "author": {
    "name": "GitHub",
    "email": "support@github.com"
  },
  "repository": {
    "type": "git",
    "url": "https://github.com/cli/cli"
  },
  "license": "MIT",
  "targets": {
    "windows-amd64": {
      "archive_type": "zip",
      "url": "https://github.com/cli/cli/releases/download/v2.87.3/gh_2.87.3_windows_amd64.zip",
      "sha256": "abc123...",
      "bin": ["bin/gh.exe"]
    },
    "linux-amd64": {
      "archive_type": "tar.gz",
      "url": "https://github.com/cli/cli/releases/download/v2.87.3/gh_2.87.3_linux_amd64.tar.gz",
      "sha256": "def456...",
      "bin": ["bin/gh"],
      "extract_root": "gh_2.87.3_linux_amd64"
    },
    "darwin-amd64": {
      "archive_type": "tar.gz",
      "url": "https://github.com/cli/cli/releases/download/v2.87.3/gh_2.87.3_darwin_amd64.tar.gz",
      "sha256": "ghi789...",
      "bin": ["bin/gh"]
    }
  }
}
```

### Required fields

- `schema`: Always set to 1
- `name`: Package identifier
- `version`: Semantic version
- `targets`: Map of platform targets (e.g., "windows-amd64", "linux-amd64")

### Target fields

- `archive_type`: "zip", "tar.gz", or "tar"
- `url`: Download URL for the archive
- `sha256`: SHA256 checksum (plain hex format. No
"SHA256:" prefix)
- `bin`: Array of relative paths to executables within the archive
- `extract_root`: (Optional) Subdirectory within archive containing the actual content

## Storage

Blaze stores downloaded packages and metadata in `~/.blaze/`:

```
~/.blaze/
├── packages/
│   └── package-name/
│       └── version/
│           ├── archive        (downloaded file)
│           ├── content        (extracted archive)
│           └── .metadata.json (binary paths for version switching)
└── registry.json              (installed packages and versions)
```

## How It Works

### Installation

1. Fetches manifest from URL
2. Selects target matching current OS and architecture
3. Downloads archive to temporary file
4. Verifies SHA256 checksum
5. Extracts to `~/.blaze/packages/{name}/{version}/`
6. Validates binaries exist
7. Adds binary directories to PATH (Windows Registry or shell profiles)
8. Saves metadata for version switching
9. Updates registry

If any step fails, the entire installation is rolled back.

### PATH Management

**Windows**: Uses Registry to update `HKCU:\Environment\PATH`

**Linux**: Appends `export PATH="..."` lines to shell profile (`~/.bashrc`, `~/.zshrc`, or `~/.profile`)

**macOS**: Similar to Linux, but untested on the OS at all. 

Changes take effect in new shell sessions only. (may vary based on platform and shell)

### Version Switching

The `use` command removes all PATH entries for the package and adds entries for the target version. This allows running multiple versions of the same tool without conflicts.

## Supported Platforms

- Windows (amd64)
- macOS (amd64, arm64)
- Linux (amd64, arm64, etc.)

Architecture aliases are supported (e.g., x64 = amd64, x86_64 = amd64).

## Error Handling

Blaze provides clear error messages for:
- Missing platform targets
- Download failures
- SHA256 mismatches
- Archive extraction errors
- Missing binaries
- PATH update failures
        
## Building from Source

```bash
go build -o blaze ./src
```

Requires Go 1.20 or later.

## Architecture

Blaze is implemented in Go for:
- Cross-platform compatibility
- Single binary distribution
- Type safety
- Built-in concurrency support

## Security

- All downloads are verified against SHA256 checksums
- Archives are extracted to isolated package directories
- No arbitrary code execution (manifests only specify binaries)
- Atomic operations prevent partial installations

## License

MIT