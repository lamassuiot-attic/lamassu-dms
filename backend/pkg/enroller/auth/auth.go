package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"

	stdjwt "github.com/dgrijalva/jwt-go"
)

const (
	publicKeyHeader = "-----BEGIN PUBLIC KEY-----"
	publicKeyFooter = "-----END PUBLIC KEY-----"
)

type Auth interface {
	Kf(token *stdjwt.Token) (interface{}, error)
	KeycloakClaimsFactory() stdjwt.Claims
}

type auth struct {
	keycloakHost     string
	keycloakPort     string
	keycloakProtocol string
	keycloakRealm    string
}

type Roles struct {
	RoleNames []string `json:"roles"`
}

type Account struct {
	roles []Roles `json:"account"`
}

type KeycloakClaims struct {
	Type                      string   `json:"typ,omitempty"`
	AuthorizedParty           string   `json:"azp,omitempty"`
	Nonce                     string   `json:"nonce,omitempty"`
	AuthTime                  int64    `json:"auth_time,omitempty"`
	SessionState              string   `json:"session_state,omitempty"`
	AuthContextClassReference string   `json:"acr,omitempty"`
	AllowedOrigins            []string `json:"allowed-origins,omitempty"`
	RealmAccess               Roles    `json:"realm_access,omitempty"`
	ResourceAccess            Account  `json:"resource_access,omitempty"`
	Scope                     string   `json:"scope,omitempty"`
	EmailVerified             bool     `json:"email_verified,omitempty"`
	FullName                  string   `json:"name,omitempty"`
	PreferredUsername         string   `json:"preferred_username,omitempty"`
	GivenName                 string   `json:"given_name,omitempty"`
	FamilyName                string   `json:"family_name,omitempty"`
	Email                     string   `json:"email,omitempty"`
	stdjwt.StandardClaims
}

var (
	errBadKey              = errors.New("Unexpected JWT key signing method")
	errBadPublicKeyRequest = errors.New("Error verifying token")
)

type KeycloakPublic struct {
	Realm           string `json:"realm"`
	PublicKey       string `json:"public_key"`
	TokenService    string `json:"token-service"`
	AccountService  string `json:"account-service"`
	TokensNotBefore int    `json:"tokens-not-before"`
}

func NewAuth(keycloakHost string, keycloakPort string, keycloakProtocol string, keycloakRealm string) Auth {
	return &auth{keycloakHost: keycloakHost, keycloakPort: keycloakPort, keycloakProtocol: keycloakProtocol, keycloakRealm: keycloakRealm}
}

func (a *auth) KeycloakClaimsFactory() stdjwt.Claims {
	return &KeycloakClaims{}
}

func (a *auth) Kf(token *stdjwt.Token) (interface{}, error) {

	if _, ok := token.Method.(*stdjwt.SigningMethodRSA); !ok {
		return nil, errBadKey
	}

	keycloakURL := a.keycloakProtocol + "://" + a.keycloakHost + ":" + a.keycloakPort + "/auth/realms/" + a.keycloakRealm
	r, err := http.Get(keycloakURL) // This should be changed in order to verify the server certificate in case HTTPS is being used.
	if err != nil {
		return nil, errBadPublicKeyRequest
	}
	var keyPublic KeycloakPublic
	if err := json.NewDecoder(r.Body).Decode(&keyPublic); err != nil {
		return nil, err
	}
	pubKey, err := parseKeycloakPublicKey([]byte(publicKeyHeader + "\n" + keyPublic.PublicKey + "\n" + publicKeyFooter))
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

func parseKeycloakPublicKey(data []byte) (*rsa.PublicKey, error) {
	pubPem, _ := pem.Decode(data)
	parsedKey, err := x509.ParsePKIXPublicKey(pubPem.Bytes)
	if err != nil {
		return nil, errors.New("Unable to parse public key")
	}
	pubKey := parsedKey.(*rsa.PublicKey)
	return pubKey, nil
}
