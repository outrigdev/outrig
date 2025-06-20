package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/outrigdev/outrig/server/pkg/serverutil"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main-generatenix.go <version>")
	}

	version := os.Args[1]
	versionClean := strings.TrimPrefix(version, "v")

	// Platform mappings for Nix
	platforms := map[string]struct {
		nixPlatform string
		goArch      string
		goPlatform  string
	}{
		"x86_64-linux":   {"x86_64-linux", "x86_64", "Linux"},
		"aarch64-linux":  {"aarch64-linux", "arm64", "Linux"},
		"x86_64-darwin":  {"x86_64-darwin", "x86_64", "Darwin"},
		"aarch64-darwin": {"aarch64-darwin", "arm64", "Darwin"},
	}

	hashes := make(map[string]string)

	for nixPlatform, platformInfo := range platforms {
		// Generate URL for the tar.gz file
		url := fmt.Sprintf("https://github.com/outrigdev/outrig/releases/download/%s/outrig_%s_%s_%s.tar.gz",
			version, versionClean, platformInfo.goPlatform, platformInfo.goArch)

		var fileData []byte
		var err error

		// Check if we should read from local files or download from GitHub
		tarFilesPath := os.Getenv("TAR_FILES_PATH")
		if tarFilesPath != "" {
			// Read from local file
			localPath := fmt.Sprintf("%s/outrig_%s_%s_%s.tar.gz", tarFilesPath, versionClean, platformInfo.goPlatform, platformInfo.goArch)
			fileData, err = os.ReadFile(localPath)
			if err != nil {
				log.Fatalf("Failed to read local file %s: %v", localPath, err)
			}
			fmt.Printf("Read local file %s: %d bytes\n", localPath, len(fileData))
		} else {
			// Download from GitHub
			fileData, err = serverutil.DownloadFile(url, 1024*1024)
			if err != nil {
				log.Fatalf("Failed to download %s: %v", url, err)
			}
			fmt.Printf("Downloaded %s: %d bytes\n", url, len(fileData))
		}

		// Calculate SHA256 hash
		hash := fmt.Sprintf("sha256:%x", sha256.Sum256(fileData))
		hashes[nixPlatform] = hash

		fmt.Printf("Generated hash for %s: %s\n", nixPlatform, hash)
	}

	// Read template
	templatePath := "./outrig-server.nix.template"
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		log.Fatalf("Failed to read template: %v", err)
	}

	// Replace placeholders
	output := string(templateContent)
	output = strings.ReplaceAll(output, "VERSION_PLACEHOLDER", versionClean)
	output = strings.ReplaceAll(output, "X86_64_LINUX_HASH_PLACEHOLDER", hashes["x86_64-linux"])
	output = strings.ReplaceAll(output, "AARCH64_LINUX_HASH_PLACEHOLDER", hashes["aarch64-linux"])
	output = strings.ReplaceAll(output, "X86_64_DARWIN_HASH_PLACEHOLDER", hashes["x86_64-darwin"])
	output = strings.ReplaceAll(output, "AARCH64_DARWIN_HASH_PLACEHOLDER", hashes["aarch64-darwin"])

	// Write output to dist directory
	outputPath := "../../../dist/outrig-server.nix"
	err = os.WriteFile(outputPath, []byte(output), 0644)
	if err != nil {
		log.Fatalf("Failed to write nix file: %v", err)
	}

	fmt.Printf("Generated Nix package for version %s\n", version)
	fmt.Printf("Output written to: %s\n", outputPath)
}
