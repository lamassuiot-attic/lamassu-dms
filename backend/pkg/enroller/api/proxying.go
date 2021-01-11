package api

import (
	"context"
	"crypto/tls"
	"device-manufacturing-system/pkg/enroller/models/csr"
	csrmodel "device-manufacturing-system/pkg/enroller/models/csr"
	"device-manufacturing-system/pkg/enroller/utils"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	consulsd "github.com/go-kit/kit/sd/consul"
	"github.com/go-kit/kit/sd/lb"
	"github.com/hashicorp/consul/api"

	httptransport "github.com/go-kit/kit/transport/http"
)

func ProxyingMiddleware(proxyURL string, proxyCA string, consulProtocol string, consulHost string, consulPort string, logger log.Logger) ServiceMiddleware {
	return func(next Service) Service {
		consulConfig := api.DefaultConfig()
		consulConfig.Address = consulProtocol + "://" + consulHost + ":" + consulPort
		consulClient, err := api.NewClient(consulConfig)
		if err != nil {
			logger.Log("err", err, "msg", "Unable to start Consul client")
			os.Exit(1)
		}
		tags := []string{"enroller", "enroller"}
		passingOnly := true
		duration := 500 * time.Millisecond
		client := consulsd.NewClient(consulClient)
		instancer := consulsd.NewInstancer(client, logger, "enroller", tags, passingOnly)

		ctx := context.Background()

		var getCSRsEndpoint, getCSRStatusEndpoint, getCRTEndpoint endpoint.Endpoint

		getCSRsFactory := makeGetCSRsFactory(ctx, "GET", proxyURL, proxyCA)
		getCSRsEndpointer := sd.NewEndpointer(instancer, getCSRsFactory, logger)
		getCSRsBalancer := lb.NewRoundRobin(getCSRsEndpointer)
		getCSRsRetry := lb.Retry(1, duration, getCSRsBalancer)
		getCSRsEndpoint = getCSRsRetry

		getCSRStatusFactory := makeGetCSRStatusFactory(ctx, "GET", proxyURL, proxyCA)
		getCSRStatusEndpointer := sd.NewEndpointer(instancer, getCSRStatusFactory, logger)
		getCSRStatusBalancer := lb.NewRoundRobin(getCSRStatusEndpointer)
		getCSRStatusRetry := lb.Retry(1, duration, getCSRStatusBalancer)
		getCSRStatusEndpoint = getCSRStatusRetry

		getCRTFactory := makeGetCRTFactory(ctx, "GET", proxyURL, proxyCA)
		getCRTEndpointer := sd.NewEndpointer(instancer, getCRTFactory, logger)
		getCRTBalancer := lb.NewRoundRobin(getCRTEndpointer)
		getCRTRetry := lb.Retry(1, duration, getCRTBalancer)
		getCRTEndpoint = getCRTRetry

		return proxymw{next, getCSRsEndpoint, getCSRStatusEndpoint, getCRTEndpoint}
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

func makeGetCSRStatusFactory(_ context.Context, method, path, proxyCA string) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		if !strings.HasPrefix(instance, "http") {
			instance = "http://" + instance
		}
		u, err := url.Parse(instance)
		if err != nil {
			panic(err)
		}
		httpc := makeProxyClient(u, proxyCA)
		options := []httptransport.ClientOption{
			httptransport.SetClient(httpc),
			httptransport.ClientBefore(jwt.ContextToHTTP()),
		}
		return httptransport.NewClient(
			method,
			u,
			encodeGetCSRStatusRequest,
			decodeGetCSRStatusResponse,
			options...,
		).Endpoint(), nil, nil
	}
}

func makeGetCSRsFactory(_ context.Context, method, path, proxyCA string) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		if !strings.HasPrefix(instance, "http") {
			instance = "http://" + instance
		}
		u, err := url.Parse(instance)
		if err != nil {
			panic(err)
		}
		httpc := makeProxyClient(u, proxyCA)
		options := []httptransport.ClientOption{
			httptransport.SetClient(httpc),
			httptransport.ClientBefore(jwt.ContextToHTTP()),
		}

		return httptransport.NewClient(
			method,
			u,
			encodeGetCSRsRequest,
			decodeGetCSRsResponse,
			options...,
		).Endpoint(), nil, nil
	}

}

func makeGetCRTFactory(_ context.Context, method, path, proxyCA string) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		if !strings.HasPrefix(instance, "http") {
			instance = "http://" + instance
		}
		u, err := url.Parse(instance)
		if err != nil {
			panic(err)
		}
		httpc := makeProxyClient(u, proxyCA)
		options := []httptransport.ClientOption{
			httptransport.SetClient(httpc),
			httptransport.ClientBefore(jwt.ContextToHTTP()),
		}

		return httptransport.NewClient(
			method,
			u,
			encodeGetCRTRequest,
			decodeGetCRTResponse,
			options...,
		).Endpoint(), nil, nil
	}

}
