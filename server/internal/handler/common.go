package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
)

func ResponseError(c *gin.Context, msg string) {
	c.Error(errors.New(msg))
	c.Abort()
}
