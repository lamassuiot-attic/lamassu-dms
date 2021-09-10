package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/tracing/opentracing"
	stdopentracing "github.com/opentracing/opentracing-go"
)

type Endpoints struct {
	HealthEndpoint        endpoint.Endpoint
	PostSetConfigEndpoint endpoint.Endpoint
	PostGetCRTEndpoint    endpoint.Endpoint
}

func MakeServerEndpoints(s Service, otTracer stdopentracing.Tracer) Endpoints {
	var healthEndpoint endpoint.Endpoint
	{
		healthEndpoint = MakeHealthEndpoint(s)
		healthEndpoint = opentracing.TraceServer(otTracer, "Health")(healthEndpoint)
	}
	var postSetConfigEndpoint endpoint.Endpoint
	{
		postSetConfigEndpoint = MakePostSetConfigEndpoint(s)
		postSetConfigEndpoint = opentracing.TraceServer(otTracer, "PostSetConfig")(postSetConfigEndpoint)
	}
	var postGetCRTEndpoint endpoint.Endpoint
	{
		postGetCRTEndpoint = MakePostGetCRTEndpoint(s)
		postGetCRTEndpoint = opentracing.TraceServer(otTracer, "PostGetCRT")(postGetCRTEndpoint)
	}
	return Endpoints{
		HealthEndpoint:        healthEndpoint,
		PostSetConfigEndpoint: postSetConfigEndpoint,
		PostGetCRTEndpoint:    postGetCRTEndpoint,
	}
}

func MakeHealthEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		healthy := s.Health(ctx)
		return healthResponse{Healthy: healthy}, nil
	}
}

func MakePostSetConfigEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(postSetConfigRequest)
		err = s.PostSetConfig(ctx, req.AuthCRT, req.CA)
		return postSetConfigResponse{Err: err}, nil
	}
}

func MakePostGetCRTEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(postGetCRTRequest)
		data, err := s.PostGetCRT(ctx, req.KeyAlg, req.KeySize, req.C, req.ST, req.L, req.O, req.OU, req.CN, req.EMAIL, req.DeviceID, "")
		return postGetCRTResponse{Data: data, Err: err}, nil
	}
}

type healthRequest struct{}

type healthResponse struct {
	Healthy bool  `json:"healthy,omitempty"`
	Err     error `json:"err,omitempty"`
}

type postSetConfigRequest struct {
	AuthCRT string `json:"crt"`
	CA      string `json:"ca"`
}

type postSetConfigExtRequest struct {
	CA string `json:"ca"`
}

type postSetConfigResponse struct {
	Err error `json:"error,omitempty"`
}

func (r postSetConfigResponse) error() error { return r.Err }

type postGetCRTRequest struct {
	KeyAlg   string `json:"keyAlg"`
	KeySize  int    `json:"keySize"`
	C        string `json:"c"`
	ST       string `json:"string"`
	L        string `json:"l"`
	O        string `json:"o"`
	OU       string `json:"ou"`
	CN       string `json:"cn"`
	EMAIL    string `json:"email"`
	DeviceID string `json:"device_id"`
	CaName   string `json:"ca_name"`
}

type postGetCRTResponse struct {
	Data []byte `json:"crt"`
	Err  error  `json:"error,omitempty"`
}

func (r postGetCRTResponse) error() error { return r.Err }
