package middleware

import (
	"log/slog"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func Recovery(log *slog.Logger, onPanic func(c *gin.Context)) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.ErrorContext(c.Request.Context(), "panic recovered",
					slog.String("request_id", GetRequestID(c)),
					slog.Any("panic", rec),
					slog.String("stack", string(debug.Stack())),
				)
				if c.Writer.Written() {
					c.Abort()
					return
				}
				onPanic(c)
			}
		}()
		c.Next()
	}
}
