module github.com/lamassuiot/device-manufacturing-system

go 1.15

require (
	github.com/HdrHistogram/hdrhistogram-go v1.1.0 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/go-kit/kit v0.10.0
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/consul/api v1.3.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/lamassuiot/lamassu-est v0.0.5
	github.com/micromdm/scep v1.0.0
	github.com/nvellon/hal v0.3.0
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.8.0
	github.com/uber/jaeger-client-go v2.25.0+incompatible
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897 // indirect
)

replace github.com/micromdm/scep => github.com/lamassuiot/scep v1.0.1-0.20210316084701-d4decbf7937e