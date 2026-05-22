package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func InternalErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Process the request first

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			status := http.StatusInternalServerError
			code := "InternalServerError"
			message := err.Error()

			var httpError HttpError
			if errors.As(err, &httpError) {
				status = httpError.StatusCode
				code = httpError.Code
				message = httpError.Message
			}

			c.JSON(status, gin.H{
				"code":    code,
				"message": message,
			})
		}
	}
}

func OpenaiErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Process the request first

		// Check if any errors were added to the context
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			status := http.StatusInternalServerError
			code := "InternalServerError"
			message := err.Error()

			var httpError HttpError
			if errors.As(err, &httpError) {
				status = httpError.StatusCode
				code = httpError.Code
				message = httpError.Message
			}

			c.JSON(status, gin.H{
				"error": gin.H{
					"message": message,
					"type":    code,
					"code":    code,
				},
			})
		}
	}
}

func AnthropicErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Process the request first

		// Check if any errors were added to the context
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			status := http.StatusInternalServerError
			code := "InternalServerError"
			message := err.Error()

			var httpError HttpError
			if errors.As(err, &httpError) {
				status = httpError.StatusCode
				code = httpError.Code
				message = httpError.Message
			}

			c.JSON(status, gin.H{
				"type": "error",
				"error": gin.H{
					"message": message,
					"type":    code,
				},
			})
		}
	}
}

func ResponsesErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			status := http.StatusInternalServerError
			code := "api_error"
			message := err.Error()

			var httpError HttpError
			if errors.As(err, &httpError) {
				status = httpError.StatusCode
				code = httpError.Code
				message = httpError.Message
			}

			c.JSON(status, gin.H{
				"type": "error",
				"error": gin.H{
					"code":    code,
					"message": message,
				},
			})
		}
	}
}
