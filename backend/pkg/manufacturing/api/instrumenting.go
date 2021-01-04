package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
)

type instrumentingMiddleware struct {
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
	next           Service
}

func NewInstumentingMiddleware(counter metrics.Counter, latency metrics.Histogram) Middleware {
	return func(next Service) Service {
		return &instrumentingMiddleware{
			requestCount:   counter,
			requestLatency: latency,
			next:           next,
		}
	}
}

func (mw *instrumentingMiddleware) Health(ctx context.Context) bool {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "Health").Add(1)
		mw.requestLatency.With("method", "Health").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.Health(ctx)
}

func (mw *instrumentingMiddleware) PostGetCRT(ctx context.Context, keyAlg string, keySize int, c string, st string, l string, o string, ou string, cn string, email string) (data []byte, err error) {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "PostGetCRT").Add(1)
		mw.requestLatency.With("method", "PostGetCRT").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.PostGetCRT(ctx, keyAlg, keySize, c, st, l, o, ou, cn, email)
}

func (mw *instrumentingMiddleware) PostSetConfig(ctx context.Context, authCRT string, CA string) (err error) {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "PostSetConfig").Add(1)
		mw.requestLatency.With("method", "PostSetConfig").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.PostSetConfig(ctx, authCRT, CA)
}
