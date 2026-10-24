package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.ErrorContext(c.Request.Context(), "panic recovered",
					slog.String("request_id", GetRequestID(c)),
					slog.Any("panic", rec),
					slog.String("stack", string(debug.Stack())),
				)
				if !c.Writer.Written() {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
						"status":     http.StatusInternalServerError,
						"title":      "Internal Server Error",
						"request_id": GetRequestID(c),
					})
					return
				}
				c.Abort()
			}
		}()
		c.Next()
	}
}
