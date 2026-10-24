package handler

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthChecker interface {
	Name() string
	Check(ctx context.Context) error
}

type HealthHandler struct {
	checkers []HealthChecker
	timeout  time.Duration
}

func NewHealthHandler(timeout time.Duration, checkers ...HealthChecker) *HealthHandler {
	return &HealthHandler{checkers: checkers, timeout: timeout}
}

type healthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// Liveness godoc
//
//	@Summary		Liveness probe
//	@Description	Reports that the process is running. Does not check dependencies.
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	healthResponse
//	@Router			/healthz [get]
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, healthResponse{Status: "ok"})
}

// Readiness godoc
//
//	@Summary		Readiness probe
//	@Description	Reports whether all dependencies (database, cache) are reachable.
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	healthResponse
//	@Failure		503	{object}	healthResponse
//	@Router			/readyz [get]
func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		checks = make(map[string]string, len(h.checkers))
		ok     = true
	)

	for _, chk := range h.checkers {
		wg.Go(func() {
			result := "ok"
			if err := chk.Check(ctx); err != nil {
				result = "unavailable"
			}
			mu.Lock()
			defer mu.Unlock()
			checks[chk.Name()] = result
			if result != "ok" {
				ok = false
			}
		})
	}
	wg.Wait()

	if !ok {
		c.JSON(http.StatusServiceUnavailable, healthResponse{Status: "unavailable", Checks: checks})
		return
	}
	c.JSON(http.StatusOK, healthResponse{Status: "ok", Checks: checks})
}
