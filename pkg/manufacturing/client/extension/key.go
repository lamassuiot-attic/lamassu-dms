package extension

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
)

func makeKey(keyAlg string, keySize int) (crypto.PrivateKey, error) {
	var key crypto.PrivateKey
	var err error
	switch keyAlg {
	case "RSA":
		key, err = newRSAKey(keySize)
	case "EC":
		key, err = newECDSAKey(keySize)
	}
	if err != nil {
		return nil, err
	}
	return key, nil
}

// create a new RSA private key
func newRSAKey(bits int) (crypto.PrivateKey, error) {
	private, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	return private, nil
}

func newECDSAKey(bits int) (crypto.PrivateKey, error) {
	var private *ecdsa.PrivateKey
	var err error
	switch bits {
	case 256:
		private, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case 384:
		private, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)

	}
	if err != nil {
		return nil, err
	}
	return private, nil
}
