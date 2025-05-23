# Generate Homebrew Cask

This tool generates a Homebrew cask file for the Outrig macOS app DMG files.

## Usage

```bash
go run main-generatecask.go <version>
```

Example:
```bash
go run main-generatecask.go v0.6.0
```

## Environment Variables

- `DMG_FILES_PATH` (optional): Path to local DMG files. If not set, the tool will download DMG files from GitHub releases.

## How it works

1. For each architecture (amd64, arm64), the tool either:
   - Reads the DMG file from the local path specified by `DMG_FILES_PATH`
   - Downloads the DMG file from the GitHub release

2. Calculates SHA256 checksums for both DMG files

3. Generates a Homebrew cask file with:
   - Version information
   - Download URLs for both architectures
   - SHA256 checksums
   - App installation instructions
   - Cleanup instructions for uninstall

4. Writes the cask file to `../../../dist/outrig.rb`

## Output

The generated cask file follows Homebrew's cask format and includes:
- Architecture-specific download URLs and checksums
- Proper app installation
- Cleanup instructions for user data
- Livecheck configuration for automatic updates

## Integration

This tool is integrated into the GitHub Actions release workflow and can also be run locally using:

```bash
task generate:cask -- v0.6.0