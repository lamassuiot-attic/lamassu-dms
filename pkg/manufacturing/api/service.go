package api

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/lamassuiot/device-manufacturing-system/pkg/manufacturing/client"
	"github.com/lamassuiot/device-manufacturing-system/pkg/manufacturing/utils"

	"github.com/pkg/errors"
)

const (
	certificatePEMBlockType = "CERTIFICATE"
	privateKeyPEMBlockType  = "PRIVATE KEY"
)

type Service interface {
	Health(ctx context.Context) bool
	PostSetConfig(ctx context.Context, authCRT string, CA string) error
	PostGetCRT(ctx context.Context, keyAlg string, keySize int, c, st, l, o, ou, cn, email, deviceId, caName string) (data []byte, err error)
}

type deviceService struct {
	mtx         sync.RWMutex
	authKeyFile string
	client      client.Client
}

func NewDeviceService(authKeyFile string, client client.Client) Service {
	return &deviceService{authKeyFile: authKeyFile, client: client}
}

var (
	//Client errors
	errGetAuthKey         = errors.New("error obtaining authentication key")
	errInvalidCert        = errors.New("invalid certificate")
	errKeyMatching        = errors.New("private and public key do not match")
	errUnsupportedKey     = errors.New("unsupported key algorithm")
	errUnsupportedECSize  = errors.New("unsupported EC key size")
	errUnsupportedRSASize = errors.New("unsupported RSA key size")
	errCNEmpty            = errors.New("invalid content, CN is required")

	//Server errors
	errRemoteConnection = errors.New("unable to start remote connection")
)

func (s *deviceService) Health(ctx context.Context) bool {
	return true
}

func (s *deviceService) PostSetConfig(ctx context.Context, authCRT string, CA string) error {
	authKey, err := loadAuthKey(s.authKeyFile)
	fmt.Println(s.authKeyFile)
	if err != nil {
		return errGetAuthKey
	}
	pemBlock, _ := pem.Decode([]byte(authCRT))
	err = utils.CheckPEMBlock(pemBlock, utils.CertPEMBlockType)
	if err != nil {
		return errInvalidCert
	}
	cert, err := tls.X509KeyPair([]byte(authCRT), authKey)
	if err != nil {
		return errKeyMatching
	}
	err = s.client.StartClient(ctx, CA, []tls.Certificate{cert})
	if err != nil {
		return errRemoteConnection
	}
	return nil
}

func (s *deviceService) PostGetCRT(ctx context.Context, keyAlg string, keySize int, c, st, l, o, ou, cn, email, deviceId, caName string) (data []byte, err error) {
	err = checkKeyAlg(keyAlg)
	if err != nil {
		return nil, err
	}

	err = checkKeySize(keyAlg, keySize)
	if err != nil {
		return nil, err
	}

	if cn == "" {
		return nil, errCNEmpty
	}

	cert, key, err := s.client.GetCertificate(ctx, keyAlg, keySize, c, st, l, o, ou, cn, email, caName)
	if err != nil {
		return nil, err
	}

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

	postBody, _ := json.Marshal(map[string]string{
		"ca_name":  caName,
		"serial_number": "lalalalalallal", //TODO: serial number comes decoding the cert
	})

	//TODO: finish this
	responseBody := bytes.NewBuffer(postBody)
	resp, err := http.Post("https://devices/v1/devices/" + deviceId + "/issue/dms/", "application/json", responseBody)
	if err != nil {
		return nil, err
	}
	fmt.Println(deviceId, resp)

	return append(utils.PEMCert(cert.Raw), utils.PEMKey(repKey)...), nil
}

func checkKeyAlg(keyAlg string) error {
	if keyAlg != "EC" && keyAlg != "RSA" {
		return errUnsupportedKey
	}
	return nil
}

// TODO: mas variedad
func checkKeySize(keyAlg string, keySize int) error {
	if keyAlg == "EC" && keySize != 384 && keySize != 256 {
		return errUnsupportedECSize
	}

	if keyAlg == "RSA" && keySize != 2048 && keySize != 4096 {
		return errUnsupportedRSASize
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
