package mocks

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
)

type MockClient struct {
	StartClientFn      func(ctx context.Context, CA string, authCRT []tls.Certificate) error
	StartClientInvoked bool

	GetCertificateFn      func(ctx context.Context, keyAlg string, keySize int, c string, st string, l string, o string, ou string, cn string, email string) (*x509.Certificate, crypto.PrivateKey, error)
	GetCertificateInvoked bool
}

func (mc *MockClient) StartClient(ctx context.Context, CA string, authCRT []tls.Certificate) error {
	mc.StartClientInvoked = true
	return mc.StartClientFn(ctx, CA, authCRT)
}

func (mc *MockClient) GetCertificate(ctx context.Context, keyAlg string, keySize int, c string, st string, l string, o string, ou string, cn string, email string) (*x509.Certificate, crypto.PrivateKey, error) {
	mc.GetCertificateInvoked = true
	return mc.GetCertificateFn(ctx, keyAlg, keySize, c, st, l, o, ou, cn, email)
}
