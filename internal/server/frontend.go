package server

import (
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func frontendDistDir() string {
	return filepath.Join("web", "dist")
}

func frontendAssetsDir() string {
	return filepath.Join(frontendDistDir(), "app-assets")
}

func serveFrontendIndex(c *gin.Context) bool {
	indexPath := filepath.Join(frontendDistDir(), "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return false
	}
	c.File(indexPath)
	return true
}
