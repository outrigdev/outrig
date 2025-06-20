# Generate Nix Package

This tool generates a Nix package file for Outrig releases by downloading the release artifacts and computing SHA256 hashes.

## Usage

```bash
# Generate from published GitHub release
go run main-generatenix.go v0.8.3

# Generate from local tar.gz files (for testing)
TAR_FILES_PATH=/path/to/tar/files go run main-generatenix.go v0.8.3
```

## Environment Variables

- `TAR_FILES_PATH`: Optional. Path to directory containing local tar.gz files for testing. If not set, files will be downloaded from GitHub releases.

## Expected File Structure

When using `TAR_FILES_PATH`, the directory should contain:
- `outrig_X.Y.Z_Linux_amd64.tar.gz`
- `outrig_X.Y.Z_Linux_arm64.tar.gz`
- `outrig_X.Y.Z_Darwin_amd64.tar.gz`
- `outrig_X.Y.Z_Darwin_arm64.tar.gz`

## Output

The tool generates `../../../dist/outrig-server.nix` with updated version and SHA256 hashes for all supported platforms.

## Integration

This tool is integrated into the release workflow via:
- Task: `task generate:nix -- v0.8.3`
- GitHub Action: `update-nix-package` job in `.github/workflows/release.yml`

The GitHub Action automatically creates a PR to update the Nix package file after a release is published.