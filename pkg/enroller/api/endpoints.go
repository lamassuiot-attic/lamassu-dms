package api

import (
	"context"
	"device-manufacturing-system/pkg/enroller/models/csr"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/tracing/opentracing"
	stdopentracing "github.com/opentracing/opentracing-go"
)

type Endpoints struct {
	HealthEndpoint       endpoint.Endpoint
	GetCSRsEndpoint      endpoint.Endpoint
	GetCSRStatusEndpoint endpoint.Endpoint
	GetCRTEndpoint       endpoint.Endpoint
}

func MakeServerEndpoints(s Service, otTracer stdopentracing.Tracer) Endpoints {
	var healthEndpoint endpoint.Endpoint
	{
		healthEndpoint = MakeHealthEndpoint(s)
		healthEndpoint = opentracing.TraceServer(otTracer, "Health")(healthEndpoint)
	}
	var getCSRsEndpoint endpoint.Endpoint
	{
		getCSRsEndpoint = MakeGetCSRsEndpoint(s)
		getCSRsEndpoint = opentracing.TraceServer(otTracer, "GetPendingCSRs")(getCSRsEndpoint)
	}
	var getCSRStatusEndpoint endpoint.Endpoint
	{
		getCSRStatusEndpoint = MakeGetCSRStatusEndpoint(s)
		getCSRStatusEndpoint = opentracing.TraceServer(otTracer, "GetPendingCSRDB")(getCSRStatusEndpoint)
	}
	var getCRTEndpoint endpoint.Endpoint
	{
		getCRTEndpoint = MakeGetCRTEndpoint(s)
		getCRTEndpoint = opentracing.TraceServer(otTracer, "GetCRT")(getCRTEndpoint)
	}
	return Endpoints{
		HealthEndpoint:       healthEndpoint,
		GetCSRsEndpoint:      getCSRsEndpoint,
		GetCSRStatusEndpoint: getCSRStatusEndpoint,
		GetCRTEndpoint:       getCRTEndpoint,
	}
}

func MakeHealthEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		healthy := s.Health(ctx)
		return healthResponse{Healthy: healthy}, nil
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

type healthRequest struct{}

type healthResponse struct {
	Healthy bool  `json:"healthy,omitempty"`
	Err     error `json:"err,omitempty"`
}

type getCSRsRequest struct{}

type getCSRsResponse struct {
	CSRs csr.CSRs `json:"csr"`
}

type getCSRsEmbeddedResponse struct {
	CSRs csr.Data `json:"_embedded"`
}

type getCSRStatusRequest struct {
	ID int
}

type getCSRStatusResponse struct {
	CSR csr.CSR `json:"-"`
	Err error   `json:"err,omitempty"`
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
