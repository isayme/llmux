package handler

import (
	"llmux/internal/config"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func isTLS(c *gin.Context) bool {
	return c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
}

func LoginHandler(c *gin.Context) {
	var req struct {
		MasterKey string `json:"master_key"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(BadRequest.WithMessage("master_key is required"))
		return
	}

	if config.Get().Server.MasterKey == "" {
		c.Error(InternalServerError.WithMessage("master key not configured"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(config.Get().Server.MasterKey), []byte(req.MasterKey)); err != nil {
		c.Error(Unauthorized.WithMessage("invalid master key"))
		return
	}

	session := sessions.Default(c)
	session.Set("authed", true)
	session.Options(sessions.Options{
		MaxAge:   config.Get().Server.Session.MaxAge,
		Path:     "/",
		HttpOnly: true,     // Prevent JavaScript access
		Secure:   isTLS(c), // Only send over HTTPS
		SameSite: http.SameSiteLaxMode,
	})
	session.Save()

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func LogoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func CheckSessionHandler(c *gin.Context) {
	session := sessions.Default(c)
	authed := session.Get("authed")

	if authed == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
	} else {
		c.JSON(http.StatusOK, gin.H{"authed": authed})
	}
}
