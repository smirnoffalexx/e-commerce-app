package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	HeaderRequestID = "X-Request-ID"
	ctxKeyRequestID = "request_id"
	maxRequestIDLen = 64
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if !isValidRequestID(id) {
			id = uuid.NewString()
		}
		c.Set(ctxKeyRequestID, id)
		c.Header(HeaderRequestID, id)
		c.Next()
	}
}

func GetRequestID(c *gin.Context) string {
	return c.GetString(ctxKeyRequestID)
}

func isValidRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}
