package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"html/template"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

//go:embed templates/*
var embeddedTemplates embed.FS

//go:embed static/dark-mode.css static/manifest.json static/offline.js static/service-worker.js
//go:embed static/icons
var embeddedStatic embed.FS

//go:embed .env.example
var envExample []byte

func parseEmbeddedTemplates() (*template.Template, error) {
	tmpl := template.New("").Funcs(template.FuncMap{
		"not": func(b bool) bool { return !b },
		"initials": func(s string) string {
			if len(s) == 0 {
				return "?"
			}
			return string([]rune(s)[:1])
		},
	})
	return tmpl.ParseFS(embeddedTemplates, "templates/*.html")
}

func extractAssets(baseDir string) error {
	err := fs.WalkDir(embeddedStatic, "static", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		// Skip user-generated dirs
		if strings.Contains(path, "uploads") || strings.Contains(path, "avatars") {
			return nil
		}
		destPath := filepath.Join(baseDir, path)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		// Always overwrite static assets (bukan skip)
		return copyEmbeddedFile(embeddedStatic, path, destPath)
	})
	if err != nil {
		log.Printf("Warning: failed to extract static: %v", err)
	}

	// Buat folder yang dibutuhkan
	for _, dir := range []string{
		filepath.Join(baseDir, "static", "uploads"),
		filepath.Join(baseDir, "static", "avatars"),
		filepath.Join(baseDir, "static", "icons"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Warning: failed to create %s: %v", dir, err)
		}
	}

	// Auto-create .env jika belum ada
	envPath := filepath.Join(baseDir, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if err := createDefaultEnv(envPath); err != nil {
			log.Printf("Warning: failed to create .env: %v", err)
		} else {
			log.Println("[INFO] .env dibuat otomatis. Silakan edit sesuai kebutuhan.")
		}
	} else {
		// .env sudah ada — pastikan JWT_SECRET tidak kosong/default
		ensureJWTSecret(envPath)
	}

	return nil
}

func createDefaultEnv(envPath string) error {
	content := string(envExample)

	// Generate JWT_SECRET acak
	secret, err := generateSecret()
	if err == nil {
		content = replaceEnvValue(content, "JWT_SECRET", secret)
	}

	// Set GIN_MODE=release
	content = replaceEnvValue(content, "GIN_MODE", "release")

	return os.WriteFile(envPath, []byte(content), 0600)
}

func ensureJWTSecret(envPath string) {
	data, err := os.ReadFile(envPath)
	if err != nil {
		return
	}
	content := string(data)

	needsUpdate := false
	if strings.Contains(content, "JWT_SECRET=\n") ||
		strings.Contains(content, "JWT_SECRET=your-super-secret") ||
		strings.Contains(content, "JWT_SECRET=\r\n") {
		secret, err := generateSecret()
		if err == nil {
			content = replaceEnvValue(content, "JWT_SECRET", secret)
			needsUpdate = true
			log.Println("[INFO] JWT_SECRET di-generate otomatis di .env")
		}
	}

	if needsUpdate {
		os.WriteFile(envPath, []byte(content), 0600)
	}
}

func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func replaceEnvValue(content, key, value string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, key+"=") {
			lines[i] = key + "=" + value
		}
	}
	return strings.Join(lines, "\n")
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
