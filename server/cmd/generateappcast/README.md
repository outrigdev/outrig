# Appcast Generator

This tool generates `appcast.xml` files for Sparkle auto-updater from the template.

## Usage

```bash
# Set the private key environment variable
export SPARKLE_PRIVATE_KEY="your_base64_encoded_private_key"

# Generate appcast.xml (downloads from GitHub releases)
go run main-generateappcast.go v0.6.0

# Or use local DMG files (for GitHub Actions)
export DMG_FILES_PATH="./path/to/dmg/files"
go run main-generateappcast.go v0.6.0
```

This tool:
- Downloads DMG files from GitHub releases OR reads from local files
- Gets real file sizes
- Generates real Ed25519 signatures using the private key
- Creates production-ready `appcast.xml`

## Environment Variables

- `SPARKLE_PRIVATE_KEY` (required): Base64-encoded Ed25519 private key for signing
- `DMG_FILES_PATH` (optional): Path to local DMG files. If set, reads from local files instead of downloading from GitHub

## Task Integration

```bash
# Generate appcast (requires SPARKLE_PRIVATE_KEY env var)
task generate:appcast -- v0.6.0
```

## GitHub Actions Integration

The generator is automatically run in the GitHub Actions workflow after the DMG files are built and uploaded to the draft release.

## Template

The generator uses `macosapp/autoupdater/appcast.xml.template` and replaces these placeholders:

- `VERSION_PLACEHOLDER` - Version number (without 'v' prefix)
- `PUBDATE_PLACEHOLDER` - Current date in RFC 2822 format
- `RELEASENOTES_PLACEHOLDER` - Empty string
- `ENCLOSURE_PLACEHOLDER` - Generated enclosure XML for both amd64 and arm64

## Output

The generated `appcast.xml` contains enclosures for both architectures:

- `Outrig-darwin-amd64-VERSION.dmg` (sparkle:arch="x86_64")
- `Outrig-darwin-arm64-VERSION.dmg` (sparkle:arch="arm64")

Each enclosure includes:
- Download URL from GitHub releases
- File size
- Ed25519 signature for verification
- Architecture and OS information