package api

import (
	"context"
	"device-manufacturing-system/pkg/enroller/models/csr"

	"github.com/go-kit/kit/endpoint"
)

type Endpoints struct {
	GetCSRsEndpoint      endpoint.Endpoint
	GetCSRStatusEndpoint endpoint.Endpoint
	GetCRTEndpoint       endpoint.Endpoint
}

func MakeServerEndpoints(s Service) Endpoints {
	return Endpoints{
		GetCSRsEndpoint:      MakeGetCSRsEndpoint(s),
		GetCSRStatusEndpoint: MakeGetCSRStatusEndpoint(s),
		GetCRTEndpoint:       MakeGetCRTEndpoint(s),
	}
}

func MakeGetCSRsEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		_ = request.(getCSRsRequest)
		csrs := s.GetCSRs(ctx)
		return getCSRsResponse{CSRs: csrs}, nil
	}
}

func MakeGetCSRStatusEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(getCSRStatusRequest)
		csr, err := s.GetCSRStatus(ctx, req.ID)
		return getCSRStatusResponse{CSR: csr, Err: err}, nil
	}
}

func MakeGetCRTEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(getCRTRequest)
		data, err := s.GetCRT(ctx, req.ID)
		return getCRTResponse{Data: data, Err: err}, nil
	}
}

type getCSRsRequest struct{}

type getCSRsResponse struct {
	CSRs csr.CSRs `json:"CSRs,omitempty"`
}

type getCSRStatusRequest struct {
	ID int
}

type getCSRStatusResponse struct {
	CSR csr.CSR `json:"CSR"`
	Err error
}

func (r getCSRStatusResponse) error() error { return r.Err }

type getCRTRequest struct {
	ID int
}

type getCRTResponse struct {
	Data []byte
	Err  error
}

func (r getCRTResponse) error() error { return r.Err }
