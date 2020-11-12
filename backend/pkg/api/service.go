package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"device-manufacturing-system/pkg/client"
	"device-manufacturing-system/pkg/utils"
	"encoding/pem"
	"io/ioutil"
	"sync"

	"github.com/pkg/errors"
)

const (
	certificatePEMBlockType = "CERTIFICATE"
	privateKeyPEMBlockType  = "PRIVATE KEY"
)

type Service interface {
	PostSetConfig(ctx context.Context, authCRT string, serverURL string) error
	PostGetCRT(ctx context.Context, keyAlg string, keySize int, c string, st string, l string, o string, ou string, cn string, email string) (data []byte, err error)
}

type deviceService struct {
	mtx         sync.RWMutex
	CAPath      string
	serverURL   string
	authKeyFile string
	client      client.Client
}

func NewDeviceService(CAPath string, authKeyFile string, client client.Client) Service {
	return &deviceService{CAPath: CAPath, authKeyFile: authKeyFile, client: client}
}

var (
	ErrCertLoading      = errors.New("unable to read certificate")
	ErrCACertLoading    = errors.New("unable to read CA certificate")
	ErrKeyMatching      = errors.New("private and public key do not match")
	ErrCertVerification = errors.New("unable to verify certificate")
	ErrGetAuthKey       = errors.New("error obtaining authentication key")
	ErrRemoteConnection = errors.New("unable to start remote connection")
)

func (s *deviceService) PostSetConfig(ctx context.Context, authCRT string, CA string) error {
	authKey, err := loadAuthKey(s.authKeyFile)
	if err != nil {
		return ErrGetAuthKey
	}
	cert, err := tls.X509KeyPair([]byte(authCRT), authKey)
	if err != nil {
		return ErrKeyMatching
	}
	err = s.client.StartRemoteClient(CA, []tls.Certificate{cert})
	if err != nil {
		return ErrRemoteConnection
	}
	return nil
}

func (s *deviceService) PostGetCRT(ctx context.Context, keyAlg string, keySize int, c string, st string, l string, o string, ou string, cn string, email string) (data []byte, err error) {
	cert, key, err := s.client.GetCertificate(keyAlg, keySize, c, st, l, o, ou, cn, email)

	var repKey []byte
	switch key.(type) {
	case *rsa.PrivateKey:
		repKey = x509.MarshalPKCS1PrivateKey(key.(*rsa.PrivateKey))
	case *ecdsa.PrivateKey:
		repKey, err = x509.MarshalECPrivateKey(key.(*ecdsa.PrivateKey))
	}
	if err != nil {
		return nil, err
	}

	return append(utils.PEMCert(cert.Raw), utils.PEMKey(repKey)...), nil
}

func (s *deviceService) authFabricationSystem(authCRT []byte) error {
	cert, err := loadFabricationSystemCert(authCRT)
	if err != nil {
		return err
	}

	caCertPool, err := createCACertPool(s.CAPath)
	if err != nil {
		return err
	}

	err = verifyFabricationSystemCert(cert, caCertPool)
	if err != nil {
		return err
	}

	return nil
}

func parseFabricationSystemCert(data []byte) ([]byte, error) {
	pemBlock, _ := pem.Decode(data)
	err := utils.CheckPEMBlock(pemBlock, utils.CertPEMBlockType)
	if err != nil {
		return nil, ErrCertVerification
	}
	return pemBlock.Bytes, nil
}

func loadFabricationSystemCert(data []byte) (*x509.Certificate, error) {
	pemBlock, _ := pem.Decode(data)
	err := utils.CheckPEMBlock(pemBlock, utils.CertPEMBlockType)
	if err != nil {
		return nil, ErrCertVerification
	}
	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return nil, ErrCertVerification
	}
	return cert, nil
}

func createCACertPool(CAPath string) (*x509.CertPool, error) {
	caCert, err := ioutil.ReadFile(CAPath)
	if err != nil {
		return nil, ErrCACertLoading
	}
	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM([]byte(caCert))
	if !ok {
		return nil, ErrCACertLoading
	}
	return caCertPool, nil
}

func verifyFabricationSystemCert(cert *x509.Certificate, caCertPool *x509.CertPool) error {
	verifyOpts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	if _, err := cert.Verify(verifyOpts); err != nil {
		return ErrCertVerification
	}
	return nil
}

func loadAuthKey(keyPath string) ([]byte, error) {
	keyPEM, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	return keyPEM, nil
}
