package server

import (
	"net/http"

	agentassets "feishu-personality-agent"
	"github.com/gin-gonic/gin"
)

func frontendAssetsFS() http.FileSystem {
	return agentassets.FrontendAssetsFS()
}

func photosFS() http.FileSystem {
	return agentassets.PhotosFS()
}

func serveFrontendIndex(c *gin.Context) bool {
	indexHTML, err := agentassets.FrontendIndex()
	if err != nil || len(indexHTML) == 0 {
		return false
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	return true
}
