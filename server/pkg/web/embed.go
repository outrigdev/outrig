// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package web

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"

	"github.com/outrigdev/outrig/server/pkg/serverbase"
)

//go:embed dist
var distFS embed.FS

// GetFileSystem returns the appropriate filesystem to use based on environment
// In development mode, it returns the local filesystem
// In production mode, it returns the embedded filesystem
func GetFileSystem() http.FileSystem {
	if serverbase.IsDev() {
		return http.Dir(".")
	}

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

	fi, err := f.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f.(io.ReadSeeker))
}
