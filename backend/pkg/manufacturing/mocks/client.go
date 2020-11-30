package mocks

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
)

type MockClient struct {
	StartRemoteClientFn      func(CA string, authCRT []tls.Certificate) error
	StartRemoteClientInvoked bool

	GetCertificateFn      func(keyAlg string, keySize int, c string, st string, l string, o string, ou string, cn string, email string) (*x509.Certificate, crypto.PrivateKey, error)
	GetCertificateInvoked bool
}

func (mc *MockClient) StartRemoteClient(CA string, authCRT []tls.Certificate) error {
	mc.StartRemoteClientInvoked = true
	return mc.StartRemoteClientFn(CA, authCRT)
}

func (mc *MockClient) GetCertificate(keyAlg string, keySize int, c string, st string, l string, o string, ou string, cn string, email string) (*x509.Certificate, crypto.PrivateKey, error) {
	mc.GetCertificateInvoked = true
	return mc.GetCertificateFn(keyAlg, keySize, c, st, l, o, ou, cn, email)
}
