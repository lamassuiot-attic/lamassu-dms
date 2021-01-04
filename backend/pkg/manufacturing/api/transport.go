package api

import (
	"context"
	"device-manufacturing-system/pkg/manufacturing/auth"
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/log"

	stdjwt "github.com/dgrijalva/jwt-go"
	"github.com/go-kit/kit/transport"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

var claims = &auth.KeycloakClaims{}

func MakeHTTPHandler(s Service, logger log.Logger, auth auth.Auth) http.Handler {
	r := mux.NewRouter()
	e := MakeServerEndpoints(s)

	options := []httptransport.ServerOption{
		httptransport.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		httptransport.ServerErrorEncoder(encodeError),
		httptransport.ServerBefore(jwt.HTTPToContext()),
	}

	r.Methods("GET").Path("/v1/device/health").Handler(httptransport.NewServer(
		e.HealthEndpoint,
		decodeHealthRequest,
		encodeResponse,
		options...,
	))

	r.Methods("POST").Path("/v1/device/config").Handler(httptransport.NewServer(
		jwt.NewParser(auth.Kf, stdjwt.SigningMethodRS256, auth.KeycloakClaimsFactory)(e.PostSetConfigEndpoint),
		decodePostSetConfigRequest,
		encodeResponse,
		options...,
	))

	r.Methods("POST").Path("/v1/device").Handler(httptransport.NewServer(
		jwt.NewParser(auth.Kf, stdjwt.SigningMethodRS256, auth.KeycloakClaimsFactory)(e.PostGetCRTEndpoint),
		decodePostGetCRTRequest,
		encodePostGetCRTResponse,
		options...,
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

func decodePostSetConfigRequest(ctx context.Context, r *http.Request) (request interface{}, err error) {
	var reqData postSetConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		return nil, err
	}
	return reqData, nil
}

func decodePostGetCRTRequest(ctx context.Context, r *http.Request) (request interface{}, err error) {
	var reqData postGetCRTRequest
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		return nil, err
	}
	return reqData, nil
}

func encodePostGetCRTResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(postGetCRTResponse)
	if resp.Err != nil {
		encodeError(ctx, resp.Err, w)
		return nil
	}
	w.Header().Set("Content-Type", "application/pkcs10; charset=utf-8")
	w.Write(resp.Data)
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
	case errGetAuthKey, errInvalidCert, errKeyMatching, errUnsupportedKey, errUnsupportedRSASize, errCNEmpty:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
