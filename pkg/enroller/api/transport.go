package api

import (
	"bytes"
	"context"
	"device-manufacturing-system/pkg/enroller/auth"
	"device-manufacturing-system/pkg/enroller/models/csr"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	stdjwt "github.com/dgrijalva/jwt-go"
	"github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/tracing/opentracing"
	"github.com/nvellon/hal"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/transport"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

func MakeHTTPHandler(s Service, logger log.Logger, auth auth.Auth, otTracer stdopentracing.Tracer) http.Handler {
	r := mux.NewRouter()
	e := MakeServerEndpoints(s, otTracer)

	options := []httptransport.ServerOption{
		httptransport.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		httptransport.ServerErrorEncoder(encodeError),
		httptransport.ServerBefore(jwt.HTTPToContext()),
	}

	r.Methods("GET").Path("/v1/health").Handler(httptransport.NewServer(
		e.HealthEndpoint,
		decodeHealthRequest,
		encodeResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(otTracer, "Health", logger)))...,
	))

	r.Methods("GET").Path("/v1/csrs").Handler(httptransport.NewServer(
		jwt.NewParser(auth.Kf, stdjwt.SigningMethodRS256, auth.KeycloakClaimsFactory)(e.GetCSRsEndpoint),
		decodeGetCSRsRequest,
		encodeGetCSRsResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(otTracer, "GetPendingCSRs", logger)))...,
	))
	r.
		Methods("GET").Path("/v1/csrs/{id}").Handler(httptransport.NewServer(
		jwt.NewParser(auth.Kf, stdjwt.SigningMethodRS256, auth.KeycloakClaimsFactory)(e.GetCSRStatusEndpoint),
		decodeGetCSRStatusRequest,
		encodeGetCSRStatusResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(otTracer, "GetPendingCSRDB", logger)))...,
	))

	r.Methods("GET").Path("/v1/csrs/{id}/crt").Handler(httptransport.NewServer(
		jwt.NewParser(auth.Kf, stdjwt.SigningMethodRS256, auth.KeycloakClaimsFactory)(e.GetCRTEndpoint),
		decodeGetCRTRequest,
		encodeGetCRTResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(otTracer, "GetPendingCSRFile", logger)))...,
	))

	return r
}

type errorer interface {
	error() error
}

func decodeHealthRequest(ctx context.Context, r *http.Request) (request interface{}, err error) {
	var req healthRequest
	return req, nil
}

func decodeGetCSRsRequest(ctx context.Context, r *http.Request) (request interface{}, err error) {
	var req getCSRsRequest
	return req, nil
}

func encodeGetCSRsResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(getCSRsResponse)
	w.Header().Set("Content-Type", "application/hal+json; charset=utf-8")
	return json.NewEncoder(w).Encode(resp)
}

func decodeGetCSRStatusRequest(ctx context.Context, r *http.Request) (request interface{}, err error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return nil, err
	}
	idNum, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	return getCSRStatusRequest{ID: idNum}, nil
}

func encodeGetCSRStatusResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(getCSRStatusResponse)
	if resp.Err != nil {
		encodeError(ctx, resp.Err, w)
		return nil
	}
	w.Header().Set("Content-Type", "application/hal+json; charset=utf-8")
	csrHal := hal.NewResource(resp.CSR, "http://localhost:8889/v1/csrs/"+strconv.Itoa(resp.CSR.Id))
	csrLink := hal.NewLink("http://localhost:8889/v1/csrs/"+strconv.Itoa(resp.CSR.Id)+"/file", hal.LinkAttr{
		"type": string("application/pkcs10"),
	})
	csrHal.AddLink("file", csrLink)
	return json.NewEncoder(w).Encode(csrHal)
}

func decodeGetCRTRequest(ctx context.Context, r *http.Request) (request interface{}, err error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return nil, err
	}
	idNum, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	return getCRTRequest{ID: idNum}, nil
}

func encodeGetCRTRequest(ctx context.Context, r *http.Request, request interface{}) error {
	req := request.(getCRTRequest)
	crtID := url.QueryEscape(strconv.Itoa(req.ID))
	r.URL.Path = "/v1/csrs/" + crtID + "/crt"
	return encodeRequest(ctx, r, request)

}

func decodeGetCRTResponse(_ context.Context, r *http.Response) (interface{}, error) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return getCRTResponse{Data: data}, nil
}

func encodeGetCRTResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(getCRTResponse)
	if resp.Err != nil {
		encodeError(ctx, resp.Err, w)
		return nil
	}
	w.Header().Set("Content-Type", "application/x-pem-file; charset=utf-8")
	w.Write(resp.Data)
	return nil
}

func encodeGetCSRsRequest(ctx context.Context, r *http.Request, request interface{}) error {
	r.URL.Path = "/v1/csrs"
	return encodeRequest(ctx, r, request)
}

func decodeGetCSRsResponse(ctx context.Context, r *http.Response) (interface{}, error) {
	var response getCSRsEmbeddedResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeGetCSRStatusRequest(ctx context.Context, r *http.Request, request interface{}) error {
	req := request.(getCSRStatusRequest)
	csrID := url.QueryEscape(strconv.Itoa(req.ID))
	r.URL.Path = "/v1/csrs/" + csrID
	return encodeRequest(ctx, r, request)
}

func decodeGetCSRStatusResponse(ctx context.Context, r *http.Response) (interface{}, error) {
	var response csr.CSR
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func encodeRequest(_ context.Context, req *http.Request, request interface{}) error {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(request)
	if err != nil {
		return err
	}
	req.Body = ioutil.NopCloser(&buf)
	return nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		// Not a Go kit transport error, but a business-logic error.
		// Provide those as HTTP errors.
		encodeError(ctx, e.error(), w)

		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	if err == nil {
		panic("encodeError with nil error")
	}
	http.Error(w, err.Error(), codeFrom(err))
}

func codeFrom(err error) int {
	switch err {
	default:
		return http.StatusInternalServerError
	}
}
