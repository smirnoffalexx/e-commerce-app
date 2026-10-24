package ratelimit

import (
	"context"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"e-commerce-backend/internal/delivery/http/middleware"
	"e-commerce-backend/internal/delivery/http/response"
)

type Limiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, remaining int, resetIn time.Duration, err error)
}

type Policy struct {
	Name     string
	Limit    int
	Window   time.Duration
	FailOpen bool
}

func Middleware(l Limiter, p Policy) gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, remaining, resetIn, err := l.Allow(c.Request.Context(), p.Name+":ip:"+c.ClientIP(), p.Limit, p.Window)
		if err != nil {
			slog.WarnContext(c.Request.Context(), "rate limiter unavailable",
				slog.String("policy", p.Name),
				slog.String("request_id", middleware.GetRequestID(c)),
				slog.Bool("fail_open", p.FailOpen),
				slog.Any("error", err),
			)
			if p.FailOpen {
				c.Next()
				return
			}
			c.Header("Retry-After", "5")
			response.Write(c, response.Problem{
				Status: http.StatusServiceUnavailable,
				Code:   "rate_limiter_unavailable",
				Detail: "service is temporarily unavailable, please retry shortly",
			})
			return
		}

		resetSeconds := strconv.Itoa(int(math.Ceil(resetIn.Seconds())))
		c.Header("RateLimit-Limit", strconv.Itoa(p.Limit))
		c.Header("RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("RateLimit-Reset", resetSeconds)

		if !allowed {
			c.Header("Retry-After", resetSeconds)
			response.Write(c, response.Problem{
				Status: http.StatusTooManyRequests,
				Code:   "rate_limited",
				Detail: "too many requests, retry after " + resetSeconds + " seconds",
			})
			return
		}
		c.Next()
	}
}
