package client

import (
	"crypto"
	"crypto/x509"
)

type Client interface {
	StartRemoteClient(CA string) error
	GetCertificate(keyAlg string, keySize int, c string, st string, l string, o string, ou string, cn string, email string) (*x509.Certificate, crypto.PrivateKey, error)
}
