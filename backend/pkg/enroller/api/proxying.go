package api

import (
	"context"
	"crypto/tls"
	csrmodel "device-manufacturing-system/pkg/enroller/models/csr"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"

	httptransport "github.com/go-kit/kit/transport/http"
)

func ProxyingMiddleware(proxyURL string, logger log.Logger) ServiceMiddleware {
	return func(next Service) Service {
		return proxymw{next, makeGetCRTProxy(proxyURL)}
	}
}

type proxymw struct {
	next   Service
	getCRT endpoint.Endpoint
}

func (mw proxymw) GetCSRs(ctx context.Context) csrmodel.CSRs {
	return mw.next.GetCSRs(ctx)
}

func (mw proxymw) GetCSRStatus(ctx context.Context, id int) (csrmodel.CSR, error) {
	return mw.next.GetCSRStatus(ctx, id)
}

func (mw proxymw) GetCRT(ctx context.Context, id int) ([]byte, error) {
	response, err := mw.getCRT(ctx, getCRTRequest{ID: id})
	if err != nil {
		return nil, err
	}
	resp := response.(getCRTResponse)
	if resp.Err != nil {
		return nil, resp.Err
	}
	return resp.Data, nil
}

func makeGetCRTProxy(proxyURL string) endpoint.Endpoint {
	if !strings.HasPrefix(proxyURL, "http") {
		proxyURL = "http://" + proxyURL
	}
	u, err := url.Parse(proxyURL)
	if err != nil {
		panic(err)
	}
	if u.Path == "" {
		u.Path = "/v1/csrs"
	}
	httpc := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	options := []httptransport.ClientOption{
		httptransport.SetClient(httpc),
		httptransport.ClientBefore(jwt.ContextToHTTP()),
	}

	return httptransport.NewClient(
		"GET",
		u,
		encodeGetCRTRequest,
		decodeGetCRTResponse,
		options...,
	).Endpoint()
}
