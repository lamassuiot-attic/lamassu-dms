package api

import (
	"context"
	"crypto/tls"
	"device-manufacturing-system/pkg/enroller/models/csr"
	csrmodel "device-manufacturing-system/pkg/enroller/models/csr"
	"device-manufacturing-system/pkg/enroller/utils"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"

	httptransport "github.com/go-kit/kit/transport/http"
)

func ProxyingMiddleware(proxyURL string, proxyCA string, logger log.Logger) ServiceMiddleware {
	return func(next Service) Service {
		return proxymw{next, makeGetCSRsProxy(proxyURL, proxyCA), makeGetCSRStatusProxy(proxyURL, proxyCA), makeGetCRTProxy(proxyURL, proxyCA)}
	}
}

type proxymw struct {
	next         Service
	getCSRs      endpoint.Endpoint
	getCSRStatus endpoint.Endpoint
	getCRT       endpoint.Endpoint
}

func (mw proxymw) Health(ctx context.Context) bool {
	return mw.next.Health(ctx)
}

func (mw proxymw) GetCSRs(ctx context.Context) csrmodel.CSRs {
	response, err := mw.getCSRs(ctx, getCSRsRequest{})
	if err != nil {
		return csrmodel.CSRs{}
	}
	resp := response.(getCSRsEmbeddedResponse)
	if resp.CSRs.EmbeddedCSRs != nil {
		return csrmodel.CSRs{CSRs: []csrmodel.CSR{resp.CSRs.EmbeddedCSRs.CSRs}}
	} else if resp.CSRs.CSRs != nil {
		return csrmodel.CSRs{CSRs: resp.CSRs.CSRs.CSRs}
	} else {
		return csrmodel.CSRs{}
	}
}

func (mw proxymw) GetCSRStatus(ctx context.Context, id int) (csrmodel.CSR, error) {
	response, err := mw.getCSRStatus(ctx, getCSRStatusRequest{ID: id})
	if err != nil {
		return csrmodel.CSR{}, err
	}
	resp := response.(csr.CSR)
	return resp, nil
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

func makeProxyClient(u *url.URL, proxyCA string) *http.Client {
	if u.Path == "" {
		u.Path = "/v1/csrs"
	}

	caCertPool, err := utils.CreateCAPool(proxyCA)
	if err != nil {
		panic(err)
	}

	httpc := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	return httpc
}

func makeGetCSRStatusProxy(proxyURL string, proxyCA string) endpoint.Endpoint {
	if !strings.HasPrefix(proxyURL, "http") {
		proxyURL = "http://" + proxyURL
	}
	u, err := url.Parse(proxyURL)
	if err != nil {
		panic(err)
	}
	httpc := makeProxyClient(u, proxyCA)
	options := []httptransport.ClientOption{
		httptransport.SetClient(httpc),
		httptransport.ClientBefore(jwt.ContextToHTTP()),
	}

	return httptransport.NewClient(
		"GET",
		u,
		encodeGetCSRStatusRequest,
		decodeGetCSRStatusResponse,
		options...,
	).Endpoint()
}

func makeGetCSRsProxy(proxyURL string, proxyCA string) endpoint.Endpoint {
	if !strings.HasPrefix(proxyURL, "http") {
		proxyURL = "http://" + proxyURL
	}
	u, err := url.Parse(proxyURL)
	if err != nil {
		panic(err)
	}
	httpc := makeProxyClient(u, proxyCA)
	options := []httptransport.ClientOption{
		httptransport.SetClient(httpc),
		httptransport.ClientBefore(jwt.ContextToHTTP()),
	}

	return httptransport.NewClient(
		"GET",
		u,
		encodeGetCSRsRequest,
		decodeGetCSRsResponse,
		options...,
	).Endpoint()
}

func makeGetCRTProxy(proxyURL string, proxyCA string) endpoint.Endpoint {
	if !strings.HasPrefix(proxyURL, "http") {
		proxyURL = "http://" + proxyURL
	}
	u, err := url.Parse(proxyURL)
	if err != nil {
		panic(err)
	}
	httpc := makeProxyClient(u, proxyCA)
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
