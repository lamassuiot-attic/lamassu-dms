package api

import (
	"context"
	"device-manufacturing-system/pkg/enroller/auth"
	csrmodel "device-manufacturing-system/pkg/enroller/models/csr"
	"device-manufacturing-system/pkg/enroller/models/csr/store"
	"errors"
	"sync"

	"github.com/go-kit/kit/auth/jwt"
)

type Service interface {
	GetCSRs(ctx context.Context) csrmodel.CSRs
	GetCSRStatus(ctx context.Context, id int) (csrmodel.CSR, error)
	GetCRT(ctx context.Context, id int) ([]byte, error)
}

type enrollerService struct {
	mtx        sync.Mutex
	csrDBStore store.DB
}

func NewEnrrolerService(csrDBStore store.DB) Service {
	return &enrollerService{csrDBStore: csrDBStore}
}

func (s *enrollerService) GetCSRs(ctx context.Context) csrmodel.CSRs {
	claims := ctx.Value(jwt.JWTClaimsContextKey).(*auth.KeycloakClaims)
	return s.csrDBStore.SelectAllByCN(claims.PreferredUsername)
}

func (s *enrollerService) GetCSRStatus(ctx context.Context, id int) (csrmodel.CSR, error) {
	csr, err := s.csrDBStore.SelectByID(id)
	if err != nil {
		return csrmodel.CSR{}, err
	}
	return csr, nil
}

func (s *enrollerService) GetCRT(ctx context.Context, id int) ([]byte, error) {
	return nil, errors.New("this method must be proxied")
}

type ServiceMiddleware func(Service) Service
