package api

import (
	"context"
	csrmodel "device-manufacturing-system/pkg/enroller/models/csr"
	"time"

	"github.com/go-kit/kit/log"
)

type Middleware func(Service) Service

func LoggingMiddleware(logger log.Logger) Middleware {
	return func(next Service) Service {
		return &loggingMiddleware{
			next:   next,
			logger: logger,
		}
	}
}

type loggingMiddleware struct {
	next   Service
	logger log.Logger
}

func (mw loggingMiddleware) GetCSRs(ctx context.Context) csrmodel.CSRs {
	defer func(begin time.Time) {
		mw.logger.Log("method", "GetCSRs", "took", time.Since(begin))
	}(time.Now())
	return mw.next.GetCSRs(ctx)
}

func (mw loggingMiddleware) GetCSRStatus(ctx context.Context, id int) (csr csrmodel.CSR, err error) {
	defer func(begin time.Time) {
		mw.logger.Log("method", "GetCSRStatus", "took", time.Since(begin), "err", err)
	}(time.Now())
	return mw.next.GetCSRStatus(ctx, id)
}

func (mw loggingMiddleware) GetCRT(ctx context.Context, id int) (data []byte, err error) {
	defer func(begin time.Time) {
		mw.logger.Log("method", "GetCRT", "took", time.Since(begin), "err", err)
	}(time.Now())
	return mw.next.GetCRT(ctx, id)
}
