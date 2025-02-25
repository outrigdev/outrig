// Copyright 2025, Outrig Inc.

package web

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
)

//go:embed dist
var distFS embed.FS

// GetFileSystem returns the appropriate filesystem to use based on environment
// In development mode, it returns the local filesystem
// In production mode, it returns the embedded filesystem
func GetFileSystem() http.FileSystem {
	// Check if we're in development mode
	if os.Getenv("OUTRIG_DEV") == "1" {
		// In development, return the local filesystem
		return http.Dir(".")
	}

	// In production, use the embedded filesystem
	distSubFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	return http.FS(distSubFS)
}

// ServeIndexOrFile serves either the requested file or falls back to index.html
// This is necessary for SPA (Single Page Application) routing
func ServeIndexOrFile(w http.ResponseWriter, r *http.Request, fs http.FileSystem) {
	// Clean the path to prevent directory traversal
	urlPath := path.Clean(r.URL.Path)

	// Try to open the file
	f, err := fs.Open(urlPath)
	if os.IsNotExist(err) {
		// If file doesn't exist, serve index.html for SPA routing
		indexFile, err := fs.Open("index.html")
		if err != nil {
			http.Error(w, "Could not find index.html", http.StatusInternalServerError)
			return
		}
		defer indexFile.Close()
		stat, err := indexFile.Stat()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, "index.html", stat.ModTime(), indexFile.(io.ReadSeeker))
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Get file info
	fi, err := f.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If it's a directory, look for index.html
	if fi.IsDir() {
		indexFile, err := fs.Open(path.Join(urlPath, "index.html"))
		if err != nil {
			http.Error(w, "Directory listing not supported", http.StatusForbidden)
			return
		}
		defer indexFile.Close()
		fi, err = indexFile.Stat()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, fi.Name(), fi.ModTime(), indexFile.(io.ReadSeeker))
		return
	}

	// Serve the file
	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f.(io.ReadSeeker))
}
