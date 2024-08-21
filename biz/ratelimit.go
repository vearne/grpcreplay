package biz

import (
	"github.com/vearne/grpcreplay/config"
	"golang.org/x/time/rate"
)

func NewRateLimit(settings *config.AppSettings) Limiter {
	if settings.RateLimitQPS > 0 {
		value := settings.RateLimitQPS
		return rate.NewLimiter(rate.Limit(value), value)
	}
	return nil
}
