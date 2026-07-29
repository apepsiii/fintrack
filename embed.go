package main

import (
	"embed"
	"html/template"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

//go:embed templates/*
var embeddedTemplates embed.FS

//go:embed static/dark-mode.css static/manifest.json static/offline.js static/service-worker.js
var embeddedStatic embed.FS

func parseEmbeddedTemplates() (*template.Template, error) {
	return template.ParseFS(embeddedTemplates, "templates/*.html")
}

func extractAssets(baseDir string) error {
	err := fs.WalkDir(embeddedStatic, "static", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		destPath := filepath.Join(baseDir, path)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		if _, err := os.Stat(destPath); err == nil {
			return nil
		}
		return copyEmbeddedFile(embeddedStatic, path, destPath)
	})
	if err != nil {
		log.Printf("Warning: failed to extract static: %v", err)
	}

	for _, dir := range []string{
		filepath.Join(baseDir, "static", "uploads"),
		filepath.Join(baseDir, "static", "avatars"),
		filepath.Join(baseDir, "static", "icons"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Warning: failed to create %s: %v", dir, err)
		}
	}

	return nil
}

func copyEmbeddedFile(fsys embed.FS, srcPath, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	src, err := fsys.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}
