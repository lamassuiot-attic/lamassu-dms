package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

type Endpoints struct {
	PostSetConfigEndpoint endpoint.Endpoint
	PostGetCRTEndpoint    endpoint.Endpoint
}

func MakeServerEndpoints(s Service) Endpoints {
	return Endpoints{
		PostSetConfigEndpoint: MakePostSetConfigEndpoint(s),
		PostGetCRTEndpoint:    MakePostGetCRTEndpoint(s),
	}
}

func MakePostSetConfigEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(postSetConfigRequest)
		err = s.PostSetConfig(ctx, req.AuthCRT, req.ServerURL)
		return postSetConfigResponse{Err: err}, nil
	}
}

func MakePostGetCRTEndpoint(s Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(postGetCRTRequest)
		data, err := s.PostGetCRT(ctx, req.KeyAlg, req.KeySize, req.C, req.ST, req.L, req.O, req.OU, req.CN, req.EMAIL)
		return postGetCRTResponse{Data: data, Err: err}, nil
	}
}

type postSetConfigRequest struct {
	AuthCRT   string `json:"crt"`
	ServerURL string `json:"serverURL"`
}

type postSetConfigResponse struct {
	Err error `json:"error,omitempty"`
}

func (r postSetConfigResponse) error() error { return r.Err }

type postGetCRTRequest struct {
	KeyAlg  string `json:"keyAlg"`
	KeySize int    `json:"keySize"`
	C       string `json:"c"`
	ST      string `json:"string"`
	L       string `json:"l"`
	O       string `json:"o"`
	OU      string `json:"ou"`
	CN      string `json:"cn"`
	EMAIL   string `json:"email"`
}

type postGetCRTResponse struct {
	Data []byte `json:"crt"`
	Err  error  `json:"error,omitempty"`
}

func (r postGetCRTResponse) error() error { return r.Err }
