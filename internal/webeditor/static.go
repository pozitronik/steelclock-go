package webeditor

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed embed/*
var embeddedAssets embed.FS

// getFileSystem returns the embedded filesystem for serving static files
func getFileSystem() (http.FileSystem, error) {
	subFS, err := fs.Sub(embeddedAssets, "embed")
	if err != nil {
		return nil, err
	}
	return http.FS(subFS), nil
}
