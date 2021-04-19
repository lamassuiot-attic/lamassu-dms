package api

import (
	"context"
	csrmodel "github.com/lamassuiot/device-manufacturing-system/pkg/enroller/models/csr"
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

func (mw loggingMiddleware) Health(ctx context.Context) (healthy bool) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "Health",
			"took", time.Since(begin),
			"healthy", healthy,
		)
	}(time.Now())
	return mw.next.Health(ctx)
}

func (mw loggingMiddleware) GetCSRs(ctx context.Context) (csrs csrmodel.CSRs) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "GetCSRs",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetCSRs(ctx)
}

func (mw loggingMiddleware) GetCSRStatus(ctx context.Context, id int) (csr csrmodel.CSR, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "GetCSRStatus",
			"request_csr_id", id,
			"response_csr_id", csr.Id,
			"csr_status", csr.Status,
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return mw.next.GetCSRStatus(ctx, id)
}

func (mw loggingMiddleware) GetCRT(ctx context.Context, id int) (data []byte, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "GetCRT",
			"request_csr_id", id,
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return mw.next.GetCRT(ctx, id)
}
