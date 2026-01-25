package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func attachFrontendRoutes(r *gin.Engine) {
	staticDir, indexPath := resolveFrontendPaths()
	if staticDir == "" || indexPath == "" {
		return
	}

	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if filePath, ok := resolveStaticFile(staticDir, c.Request.URL.Path); ok {
			c.File(filePath)
			return
		}
		c.File(indexPath)
	})
}

func resolveFrontendPaths() (string, string) {
	distDir := filepath.Join("frontend", "dist")
	if dirExists(distDir) && fileExists(filepath.Join(distDir, "index.html")) {
		return distDir, filepath.Join(distDir, "index.html")
	}

	rootDir := "frontend"
	if fileExists(filepath.Join(rootDir, "index.html")) {
		return rootDir, filepath.Join(rootDir, "index.html")
	}

	return "", ""
}

func resolveStaticFile(staticDir, urlPath string) (string, bool) {
	trimmed := strings.TrimPrefix(urlPath, "/")
	if trimmed == "" {
		return "", false
	}
	cleaned := filepath.Clean(trimmed)
	if cleaned == "." || strings.HasPrefix(cleaned, "..") {
		return "", false
	}
	filePath := filepath.Join(staticDir, cleaned)
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return "", false
	}
	return filePath, true
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
