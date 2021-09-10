package client

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
)

type Client interface {
	StartClient(ctx context.Context, CA string, authCRT []tls.Certificate) error
	GetCertificate(ctx context.Context, keyAlg string, keySize int, c string, st string, l string, o string, ou string, cn string, email string, caName string) (*x509.Certificate, crypto.PrivateKey, error)
}
