package api

import (
	"context"
	csrmodel "device-manufacturing-system/pkg/enroller/models/csr"
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

func (mw *instrumentingMiddleware) GetCSRs(ctx context.Context) (csrs csrmodel.CSRs) {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "GetCSRs").Add(1)
		mw.requestLatency.With("method", "GetCSRs").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.GetCSRs(ctx)
}

func (mw *instrumentingMiddleware) GetCSRStatus(ctx context.Context, id int) (csr csrmodel.CSR, err error) {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "GetCSRStatus").Add(1)
		mw.requestLatency.With("method", "GetCSRStatus").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.GetCSRStatus(ctx, id)
}

func (mw *instrumentingMiddleware) GetCRT(ctx context.Context, id int) (data []byte, err error) {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "GetCRT").Add(1)
		mw.requestLatency.With("method", "GetCRT").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.GetCRT(ctx, id)
}
