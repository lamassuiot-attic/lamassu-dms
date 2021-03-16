package api

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"device-manufacturing-system/pkg/manufacturing/client"
	"device-manufacturing-system/pkg/manufacturing/configs"
	"device-manufacturing-system/pkg/manufacturing/mocks"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"
	"time"
)

type serviceSetUp struct {
	authKeyFile string
	client      client.Client
}

func TestPostSetConfig(t *testing.T) {
	stu := setup(t)
	srv := NewDeviceService(stu.authKeyFile, stu.client)
	ctx := context.Background()

	stu.client.(*mocks.MockClient).StartClientFn = func(ctx context.Context, CA string, authCRT []tls.Certificate) error {
		return nil
	}

	testCases := []struct {
		name string
		cert string
		ret  error
	}{
		{"Private and public key does not match", testAuthCRT(t), errKeyMatching},
		{"Certificate is not valid", "this is not a certificate", errInvalidCert},
		{"Certificate and key are valid", loadTestAuthCRT(t), nil},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing %s", tc.name), func(t *testing.T) {
			err := srv.PostSetConfig(ctx, tc.cert, "CA")
			if tc.ret != err {
				t.Errorf("Got result is %s; want %s", err, tc.ret)
			}
		})
	}
}

func TestPostGetCRT(t *testing.T) {
	stu := setup(t)
	srv := NewDeviceService(stu.authKeyFile, stu.client)
	ctx := context.Background()

	stu.client.(*mocks.MockClient).GetCertificateFn = func(ctx context.Context, keyAlg string, keySize int, c string, st string, l string, o string, ou string, cn string, email string) (*x509.Certificate, crypto.PrivateKey, error) {
		key, err := testSCEPKey(keyAlg, keySize)
		if err != nil {
			return nil, nil, err
		}

		cert, err := testSCEPCert(key)
		if err != nil {
			return nil, nil, err
		}

		return cert, key, nil
	}

	testCases := []struct {
		name    string
		keyAlg  string
		keySize int
		cn      string
		ret     error
	}{
		{"Key Algorithm is unsupported", "unsupportedAlg", 1024, "test", errUnsupportedKey},
		{"EC Key size is unsupported", "EC", 2048, "test", errUnsupportedECSize},
		{"RSA Key size is unsupported", "RSA", 1024, "test", errUnsupportedRSASize},
		{"CN is empty", "RSA", 2048, "", errCNEmpty},
		{"EC Key, size and CN are valid", "EC", 256, "test", nil},
		{"RSA Key, size, and CN are valid", "RSA", 2048, "test", nil},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing %s", tc.name), func(t *testing.T) {
			_, err := srv.PostGetCRT(ctx, tc.keyAlg, tc.keySize, "", "", "", "", "", tc.cn, "")
			if tc.ret != err {
				t.Errorf("Got result is %s; want %s", err, tc.ret)
			}
		})
	}
}

func setup(t *testing.T) *serviceSetUp {
	t.Helper()

	cfg, err := configs.NewConfig("manufacturingtest")
	if err != nil {
		t.Fatal("Unable to get configuration variables")
	}
	client := &mocks.MockClient{}

	return &serviceSetUp{authKeyFile: cfg.AuthKeyFile, client: client}
}

func loadTestAuthCRT(t *testing.T) string {
	certPEM, err := ioutil.ReadFile("testdata/test.crt")
	if err != nil {
		t.Fatal("Unable to load test certificate")
	}
	return string(certPEM)
}

func testAuthCRT(t *testing.T) string {
	t.Helper()

	key, _ := rsa.GenerateKey(rand.Reader, 2048)

	subj := pkix.Name{
		CommonName:   "test.com",
		Country:      []string{"ES"},
		Province:     []string{"Gipuzkoa"},
		Locality:     []string{"Arrasate"},
		Organization: []string{"Test"},
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               subj,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, key.Public(), key)
	if err != nil {
		t.Fatal("Failed to create test certificate")
	}

	out := &bytes.Buffer{}
	pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	return out.String()
}

func testSCEPKey(keyAlg string, keySize int) (crypto.PrivateKey, error) {
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

func testSCEPCert(key crypto.PrivateKey) (*x509.Certificate, error) {
	subj := pkix.Name{
		CommonName:   "test.com",
		Country:      []string{"ES"},
		Province:     []string{"Gipuzkoa"},
		Locality:     []string{"Arrasate"},
		Organization: []string{"Test"},
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               subj,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	var derBytes []byte
	var err error
	switch key.(type) {
	case *rsa.PrivateKey:
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, &template, key.(*rsa.PrivateKey).Public(), key)
	case *ecdsa.PrivateKey:
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, &template, key.(*ecdsa.PrivateKey).Public(), key)
	}
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, err
	}

	return cert, nil
}
