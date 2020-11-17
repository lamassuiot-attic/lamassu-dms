package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
)

type Middleware func(Service) Service

func LoggingMidleware(logger log.Logger) Middleware {
	return func(next Service) Service {
		return &loggingMidleware{
			next:   next,
			logger: logger,
		}
	}
}

type loggingMidleware struct {
	next   Service
	logger log.Logger
}

func (mw loggingMidleware) PostGetCRT(ctx context.Context, keyAlg string, keySize int, c string, st string, l string, o string, ou string, cn string, email string) (data []byte, err error) {
	defer func(begin time.Time) {
		mw.logger.Log("method", "PostGetCRT", "took", time.Since(begin), "err", err)
	}(time.Now())
	return mw.next.PostGetCRT(ctx, keyAlg, keySize, c, st, l, o, ou, cn, email)
}

func (mw loggingMidleware) PostSetConfig(ctx context.Context, authCRT string, CA string) (err error) {
	defer func(begin time.Time) {
		mw.logger.Log("method", "PostGetCRT", "took", time.Since(begin), "err", err)
	}(time.Now())
	return mw.next.PostSetConfig(ctx, authCRT, CA)
}
