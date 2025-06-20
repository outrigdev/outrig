package serverutil

import (
	"fmt"
	"io"
	"net/http"
)

// DownloadFile downloads a file from the given URL and returns its contents.
// It performs basic validation including a minimum size check.
func DownloadFile(url string, minSize int) ([]byte, error) {
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

	// Sanity check - validate minimum file size
	if len(fileData) < minSize {
		return nil, fmt.Errorf("file too small (%d bytes), expected at least %d bytes", len(fileData), minSize)
	}

	return fileData, nil
}