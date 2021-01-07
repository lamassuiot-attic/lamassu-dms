package api

import (
	"context"
	csrmodel "device-manufacturing-system/pkg/enroller/models/csr"
	"errors"
	"sync"
)

type Service interface {
	Health(ctx context.Context) bool
	GetCSRs(ctx context.Context) csrmodel.CSRs
	GetCSRStatus(ctx context.Context, id int) (csrmodel.CSR, error)
	GetCRT(ctx context.Context, id int) ([]byte, error)
}

type enrollerService struct {
	mtx sync.Mutex
}

func NewEnrrolerService() Service {
	return &enrollerService{}
}

func (s *enrollerService) Health(ctx context.Context) bool {
	return true
}

func (s *enrollerService) GetCSRs(ctx context.Context) csrmodel.CSRs {
	return csrmodel.CSRs{}
}

func (s *enrollerService) GetCSRStatus(ctx context.Context, id int) (csrmodel.CSR, error) {
	return csrmodel.CSR{}, errors.New("this method must be proxied")
}

func (s *enrollerService) GetCRT(ctx context.Context, id int) ([]byte, error) {
	return nil, errors.New("this method must be proxied")
}

type ServiceMiddleware func(Service) Service
