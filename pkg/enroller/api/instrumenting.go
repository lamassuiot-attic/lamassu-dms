package api

import (
	"context"
	"fmt"
	csrmodel "github.com/lamassuiot/device-manufacturing-system/pkg/enroller/models/csr"
	"time"

	"github.com/go-kit/kit/metrics"
)

type instrumentingMiddleware struct {
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
	next           Service
}

func NewInstrumentingMiddleware(counter metrics.Counter, latency metrics.Histogram) Middleware {
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
		lvs := []string{"method", "Health", "error", "false"}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.Health(ctx)
}

func (mw *instrumentingMiddleware) GetCSRs(ctx context.Context) (csrs csrmodel.CSRs) {
	defer func(begin time.Time) {
		lvs := []string{"method", "GetCSRs", "error", "false"}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.GetCSRs(ctx)
}

func (mw *instrumentingMiddleware) GetCSRStatus(ctx context.Context, id int) (csr csrmodel.CSR, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "GetCSRStatus", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.GetCSRStatus(ctx, id)
}

func (mw *instrumentingMiddleware) GetCRT(ctx context.Context, id int) (data []byte, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "GetCRT", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.GetCRT(ctx, id)
}
