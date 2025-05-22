package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type EnclosureData struct {
	URL           string
	Version       string
	OS            string
	Arch          string
	Length        int64
	EdSignature   string
}

type AppcastData struct {
	Version      string
	PubDate      string
	ReleaseNotes string
	Enclosures   []EnclosureData
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main-generateappcast.go <version>")
	}

	version := os.Args[1]

	// Get private key from environment
	privateKeyB64 := os.Getenv("SPARKLE_PRIVATE_KEY")
	if privateKeyB64 == "" {
		log.Fatal("SPARKLE_PRIVATE_KEY environment variable not set")
	}

	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		log.Fatalf("Failed to decode private key: %v", err)
	}

	var privateKey ed25519.PrivateKey
	if len(privateKeyBytes) == ed25519.SeedSize {
		// Sparkle format: 32-byte seed, need to derive full private key
		privateKey = ed25519.NewKeyFromSeed(privateKeyBytes)
	} else if len(privateKeyBytes) == ed25519.PrivateKeySize {
		// Full 64-byte private key
		privateKey = ed25519.PrivateKey(privateKeyBytes)
	} else {
		log.Fatalf("Invalid private key size: expected %d (seed) or %d (full key), got %d", ed25519.SeedSize, ed25519.PrivateKeySize, len(privateKeyBytes))
	}

	// Generate enclosures for both architectures
	enclosures := []EnclosureData{}
	
	for _, arch := range []string{"amd64", "arm64"} {
		dmgURL := fmt.Sprintf("https://github.com/outrigdev/outrig/releases/download/%s/Outrig-darwin-%s-%s.dmg", version, arch, version)
		
		var fileData []byte
		var err error
		
		// Check if we should read from local files or download from GitHub
		dmgFilesPath := os.Getenv("DMG_FILES_PATH")
		if dmgFilesPath != "" {
			// Read from local file
			localPath := fmt.Sprintf("%s/Outrig-%s.dmg", dmgFilesPath, arch)
			fileData, err = os.ReadFile(localPath)
			if err != nil {
				log.Fatalf("Failed to read local file %s: %v", localPath, err)
			}
			fmt.Printf("Read local file %s: %d bytes\n", localPath, len(fileData))
		} else {
			// Download from GitHub
			fileData, err = downloadFile(dmgURL)
			if err != nil {
				log.Fatalf("Failed to download %s: %v", dmgURL, err)
			}
			fmt.Printf("Downloaded %s: %d bytes\n", dmgURL, len(fileData))
		}

		// Generate signature
		signature := ed25519.Sign(privateKey, fileData)
		signatureB64 := base64.StdEncoding.EncodeToString(signature)

		sparkleArch := arch
		if arch == "amd64" {
			sparkleArch = "x86_64"
		}

		enclosure := EnclosureData{
			URL:         dmgURL,
			Version:     strings.TrimPrefix(version, "v"),
			OS:          "macos",
			Arch:        sparkleArch,
			Length:      int64(len(fileData)),
			EdSignature: signatureB64,
		}
		enclosures = append(enclosures, enclosure)
		
		fmt.Printf("Generated signature for %s architecture\n", arch)
	}

	// Read template
	templatePath := "../../../macosapp/autoupdater/appcast.xml.template"
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		log.Fatalf("Failed to read template: %v", err)
	}

	// Generate enclosure XML
	enclosureXML := ""
	for _, enc := range enclosures {
		enclosureXML += fmt.Sprintf(`            <enclosure
                url="%s"
                sparkle:version="%s"
                sparkle:shortVersionString="%s"
                sparkle:os="%s"
                sparkle:arch="%s"
                length="%d"
                type="application/x-apple-diskimage"
                sparkle:edSignature="%s" />
`, enc.URL, enc.Version, enc.Version, enc.OS, enc.Arch, enc.Length, enc.EdSignature)
	}

	// Replace placeholders
	output := string(templateContent)
	output = strings.ReplaceAll(output, "VERSION_PLACEHOLDER", strings.TrimPrefix(version, "v"))
	output = strings.ReplaceAll(output, "PUBDATE_PLACEHOLDER", time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700"))
	output = strings.ReplaceAll(output, "RELEASENOTES_PLACEHOLDER", "")
	output = strings.ReplaceAll(output, "ENCLOSURE_PLACEHOLDER", enclosureXML)

	// Write output to dist directory (3 levels up from server/cmd/generateappcast)
	outputPath := "../../../dist/appcast.xml"
	err = os.WriteFile(outputPath, []byte(output), 0644)
	if err != nil {
		log.Fatalf("Failed to write appcast.xml: %v", err)
	}

	fmt.Printf("Generated appcast.xml for version %s\n", version)
	fmt.Printf("Output written to: %s\n", outputPath)
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