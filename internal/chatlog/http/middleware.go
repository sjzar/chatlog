package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sjzar/chatlog/internal/chatlog/database"
)

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func (s *Service) checkDBStateMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch s.db.State {
		case database.StateInit:
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database is not ready"})
			c.Abort()
			return
		case database.StateDecrypting:
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database is decrypting, please wait"})
			c.Abort()
			return
		case database.StateError:
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database is error: " + s.db.StateMsg})
			c.Abort()
			return
		}

		c.Next()
	}
}
