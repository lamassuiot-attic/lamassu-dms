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
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/sd"
	consulsd "github.com/go-kit/kit/sd/consul"
	"github.com/go-kit/kit/sd/lb"
	"github.com/go-kit/kit/tracing/opentracing"
	"github.com/hashicorp/consul/api"
	stdopentracing "github.com/opentracing/opentracing-go"

	httptransport "github.com/go-kit/kit/transport/http"
)

func ProxyingMiddleware(proxyURL string, proxyCA string, consulProtocol string, consulHost string, consulPort string, consulCA string, logger log.Logger, otTracer stdopentracing.Tracer) ServiceMiddleware {
	return func(next Service) Service {
		consulConfig := api.DefaultConfig()
		consulConfig.Address = consulProtocol + "://" + consulHost + ":" + consulPort
		tlsConf := &api.TLSConfig{CAFile: consulCA}
		consulConfig.TLSConfig = *tlsConf
		consulClient, err := api.NewClient(consulConfig)
		if err != nil {
			level.Error(logger).Log("err", err, "msg", "Could not start Consul API Client")
			os.Exit(1)
		}
		tags := []string{"enroller", "enroller"}
		passingOnly := true
		duration := 500 * time.Millisecond
		client := consulsd.NewClient(consulClient)
		instancer := consulsd.NewInstancer(client, logger, "enroller", tags, passingOnly)

		var getCSRsEndpoint, getCSRStatusEndpoint, getCRTEndpoint endpoint.Endpoint

		getCSRsFactory := makeGetCSRsFactory("GET", proxyURL, proxyCA, logger, otTracer)
		getCSRsEndpointer := sd.NewEndpointer(instancer, getCSRsFactory, logger)
		getCSRsBalancer := lb.NewRoundRobin(getCSRsEndpointer)
		getCSRsRetry := lb.Retry(1, duration, getCSRsBalancer)
		getCSRsEndpoint = getCSRsRetry
		getCSRsEndpoint = opentracing.TraceClient(otTracer, "GetPendingCSRs")(getCSRsEndpoint)

		getCSRStatusFactory := makeGetCSRStatusFactory("GET", proxyURL, proxyCA, logger, otTracer)
		getCSRStatusEndpointer := sd.NewEndpointer(instancer, getCSRStatusFactory, logger)
		getCSRStatusBalancer := lb.NewRoundRobin(getCSRStatusEndpointer)
		getCSRStatusRetry := lb.Retry(1, duration, getCSRStatusBalancer)
		getCSRStatusEndpoint = getCSRStatusRetry
		getCSRStatusEndpoint = opentracing.TraceClient(otTracer, "GetPendingCSRDB")(getCSRStatusEndpoint)

		getCRTFactory := makeGetCRTFactory("GET", proxyURL, proxyCA, logger, otTracer)
		getCRTEndpointer := sd.NewEndpointer(instancer, getCRTFactory, logger)
		getCRTBalancer := lb.NewRoundRobin(getCRTEndpointer)
		getCRTRetry := lb.Retry(1, duration, getCRTBalancer)
		getCRTEndpoint = getCRTRetry
		getCRTEndpoint = opentracing.TraceClient(otTracer, "GetCRT")(getCRTEndpoint)

		return proxymw{next, logger, getCSRsEndpoint, getCSRStatusEndpoint, getCRTEndpoint}
	}
}

type proxymw struct {
	next         Service
	logger       log.Logger
	getCSRs      endpoint.Endpoint
	getCSRStatus endpoint.Endpoint
	getCRT       endpoint.Endpoint
}

func (mw proxymw) Health(ctx context.Context) bool {
	return mw.next.Health(ctx)
}

func (mw proxymw) GetCSRs(ctx context.Context) csrmodel.CSRs {
	level.Info(mw.logger).Log("msg", "Proxying GetCSRs request to Enroller")
	response, err := mw.getCSRs(ctx, getCSRsRequest{})
	if err != nil {
		level.Error(mw.logger).Log("err", err, "msg", "Error proxying GetCSRs request to Enroller")
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
	level.Info(mw.logger).Log("msg", "Proxying GetCSRStatus request to Enroller")
	response, err := mw.getCSRStatus(ctx, getCSRStatusRequest{ID: id})
	if err != nil {
		level.Error(mw.logger).Log("err", err, "msg", "Error proxying GetCSRStatus request to Enroller")
		return csrmodel.CSR{}, err
	}
	resp := response.(csr.CSR)
	return resp, nil
}

func (mw proxymw) GetCRT(ctx context.Context, id int) ([]byte, error) {
	level.Info(mw.logger).Log("msg", "Proxying GetCRT request to Enroller")
	response, err := mw.getCRT(ctx, getCRTRequest{ID: id})
	if err != nil {
		level.Error(mw.logger).Log("err", err, "msg", "Error proxying GetCRT request to Enroller")
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

func makeGetCSRStatusFactory(method, path, proxyCA string, logger log.Logger, otTracer stdopentracing.Tracer) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		if !strings.HasPrefix(instance, "https") {
			instance = "https://" + instance
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
			append(options, httptransport.ClientBefore(opentracing.ContextToHTTP(otTracer, logger)))...,
		).Endpoint(), nil, nil
	}
}

func makeGetCSRsFactory(method, path, proxyCA string, logger log.Logger, otTracer stdopentracing.Tracer) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		if !strings.HasPrefix(instance, "http") {
			instance = "https://" + instance
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
			append(options, httptransport.ClientBefore(opentracing.ContextToHTTP(otTracer, logger)))...,
		).Endpoint(), nil, nil
	}

}

func makeGetCRTFactory(method, path, proxyCA string, logger log.Logger, otTracer stdopentracing.Tracer) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		if !strings.HasPrefix(instance, "http") {
			instance = "https://" + instance
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
			append(options, httptransport.ClientBefore(opentracing.ContextToHTTP(otTracer, logger)))...,
		).Endpoint(), nil, nil
	}

}
