module device-manufacturing-system

go 1.15

require (
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/go-kit/kit v0.10.0
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/consul/api v1.3.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/lib/pq v1.9.0
	github.com/micromdm/scep v1.0.0
	github.com/nvellon/hal v0.3.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.8.0
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
)

replace github.com/micromdm/scep => /home/mamuchastegui/go/src/github.com/micromdm/scep

replace github.com/fullsailor/pkcs7 => github.com/groob/pkcs7 v0.0.0-20180824154052-36585635cb64
