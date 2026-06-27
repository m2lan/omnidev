package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// NotImplemented returns a 501 Not Implemented handler.
// Used as a placeholder for routes that haven't been built yet.
func NotImplemented(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": gin.H{
				"code":    501,
				"message": "not implemented",
				"detail":  feature + " is not yet implemented",
			},
		})
	}
}
