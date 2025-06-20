# Generate Nix Package

This tool generates a Nix package file for Outrig releases by downloading the release artifacts and computing SHA256 hashes. It uses the `outrig-server.nix.template` file as a base and replaces placeholders with actual version numbers and SHA256 hashes.

## Usage

**Important**: This tool must be run from the `server/cmd/generatenix` directory since it looks for the template file in the current working directory.

```bash
# Change to the correct directory first
cd server/cmd/generatenix

# Generate from published GitHub release (for testing)
go run main-generatenix.go v0.8.3

# Generate from local tar.gz files (for production releases)
TAR_FILES_PATH=/path/to/tar/files go run main-generatenix.go v0.8.3
```

## Environment Variables

- `TAR_FILES_PATH`: Optional. Path to directory containing local tar.gz files created by goreleaser. Used in production releases to calculate SHA256 sums from the newly created tar.gz files. If not set, files will be downloaded from GitHub releases (only works for already published releases).

## Expected File Structure

When using `TAR_FILES_PATH`, the directory should contain:
- `outrig_X.Y.Z_Linux_amd64.tar.gz`
- `outrig_X.Y.Z_Linux_arm64.tar.gz`
- `outrig_X.Y.Z_Darwin_amd64.tar.gz`
- `outrig_X.Y.Z_Darwin_arm64.tar.gz`

## Template File

The tool uses `outrig-server.nix.template` as the base template and replaces the following placeholders:
- `VERSION_PLACEHOLDER`: Replaced with the actual version number
- `X86_64_LINUX_HASH_PLACEHOLDER`: SHA256 hash for Linux x86_64
- `AARCH64_LINUX_HASH_PLACEHOLDER`: SHA256 hash for Linux ARM64
- `X86_64_DARWIN_HASH_PLACEHOLDER`: SHA256 hash for macOS x86_64
- `AARCH64_DARWIN_HASH_PLACEHOLDER`: SHA256 hash for macOS ARM64

## Output

The tool generates `../../../dist/outrig-server.nix` (the project root's `dist` directory, assuming the tool is run from the correct working directory) with updated version and SHA256 hashes for all supported platforms.

## Integration

This tool is integrated into the release workflow via:
- Task: `task generate:nix -- v0.8.3`
- GitHub Action: `update-nix-package` job in `.github/workflows/release.yml`

The GitHub Action uses `TAR_FILES_PATH` to calculate SHA256 sums from the tar.gz files created by goreleaser during the release process, then automatically creates a PR to update the Nix package file.