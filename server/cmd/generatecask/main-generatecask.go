package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type CaskData struct {
	Version     string
	AMD64URL    string
	ARM64URL    string
	AMD64SHA256 string
	ARM64SHA256 string
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main-generatecask.go <version>")
	}

	version := os.Args[1]
	versionClean := strings.TrimPrefix(version, "v")

	// Generate URLs for both architectures
	amd64URL := fmt.Sprintf("https://github.com/outrigdev/outrig/releases/download/%s/Outrig-darwin-amd64-%s.dmg", version, version)
	arm64URL := fmt.Sprintf("https://github.com/outrigdev/outrig/releases/download/%s/Outrig-darwin-arm64-%s.dmg", version, version)

	var amd64Data, arm64Data []byte
	var err error

	// Check if we should read from local files or download from GitHub
	dmgFilesPath := os.Getenv("DMG_FILES_PATH")
	if dmgFilesPath != "" {
		// Read from local files
		amd64Path := fmt.Sprintf("%s/Outrig-amd64.dmg", dmgFilesPath)
		arm64Path := fmt.Sprintf("%s/Outrig-arm64.dmg", dmgFilesPath)
		
		amd64Data, err = os.ReadFile(amd64Path)
		if err != nil {
			log.Fatalf("Failed to read local file %s: %v", amd64Path, err)
		}
		
		arm64Data, err = os.ReadFile(arm64Path)
		if err != nil {
			log.Fatalf("Failed to read local file %s: %v", arm64Path, err)
		}
		
		fmt.Printf("Read local files: amd64=%d bytes, arm64=%d bytes\n", len(amd64Data), len(arm64Data))
	} else {
		// Download from GitHub
		amd64Data, err = downloadFile(amd64URL)
		if err != nil {
			log.Fatalf("Failed to download %s: %v", amd64URL, err)
		}
		
		arm64Data, err = downloadFile(arm64URL)
		if err != nil {
			log.Fatalf("Failed to download %s: %v", arm64URL, err)
		}
		
		fmt.Printf("Downloaded files: amd64=%d bytes, arm64=%d bytes\n", len(amd64Data), len(arm64Data))
	}

	// Calculate SHA256 checksums
	amd64Hash := fmt.Sprintf("%x", sha256.Sum256(amd64Data))
	arm64Hash := fmt.Sprintf("%x", sha256.Sum256(arm64Data))

	caskData := CaskData{
		Version:     versionClean,
		AMD64URL:    amd64URL,
		ARM64URL:    arm64URL,
		AMD64SHA256: amd64Hash,
		ARM64SHA256: arm64Hash,
	}

	// Generate cask content
	caskContent := generateCaskContent(caskData)

	// Write output to dist directory
	outputPath := "../../../dist/outrig.rb"
	err = os.WriteFile(outputPath, []byte(caskContent), 0644)
	if err != nil {
		log.Fatalf("Failed to write cask file: %v", err)
	}

	fmt.Printf("Generated Homebrew cask for version %s\n", version)
	fmt.Printf("Output written to: %s\n", outputPath)
}

func generateCaskContent(data CaskData) string {
	return fmt.Sprintf(`cask "outrig" do
  version "%s"
  
  on_intel do
    url "%s"
    sha256 "%s"
  end
  
  on_arm do
    url "%s"
    sha256 "%s"
  end

  name "Outrig"
  desc "Real-time debugging for Go programs, similar to Chrome DevTools"
  homepage "https://github.com/outrigdev/outrig"

  auto_updates true

  livecheck do
    url "https://github.com/outrigdev/outrig/releases/latest"
    strategy :github_latest
  end

  app "Outrig.app"

  zap trash: [
    "~/Library/Application Support/Outrig",
    "~/Library/Caches/Outrig",
    "~/Library/Logs/Outrig",
    "~/Library/Preferences/com.outrig.Outrig.plist",
  ]
end
`, data.Version, data.AMD64URL, data.AMD64SHA256, data.ARM64URL, data.ARM64SHA256)
}

func downloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d when downloading %s", resp.StatusCode, url)
	}

	// Read file contents
	fileData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %v", err)
	}

	// Sanity check - DMG files should be at least 1MB
	if len(fileData) < 1024*1024 {
		return nil, fmt.Errorf("file too small (%d bytes), expected at least 1MB", len(fileData))
	}

	return fileData, nil
}