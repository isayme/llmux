package internal

import "github.com/gin-gonic/gin"

// Name name of the app
var Name = "llmux"

// Version version of the app, set by ldflags
var Version = "unknown"

func VersionHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"name":    Name,
		"version": Version,
	})
}
