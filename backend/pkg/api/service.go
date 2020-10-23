package api

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"device-manufacturing-system/crypto/x509util"

	scepclient "github.com/micromdm/scep/client"
	"github.com/micromdm/scep/scep"
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
	CertFile    string
	KeyFile     string
	serverURL   string
	SCEPMapping map[string]string
}

func NewDeviceService(CAPath string, CertFile string, KeyFile string, SCEPMapping map[string]string) Service {
	return &deviceService{CAPath: CAPath, CertFile: CertFile, KeyFile: KeyFile, SCEPMapping: SCEPMapping}
}

var (
	ErrBadRequest = errors.New("Bad Request")
)

func (s *deviceService) PostSetConfig(ctx context.Context, authCRT string, CA string) error {
	err := s.authFabricationSystem(authCRT)
	if err != nil {
		return err
	}
	s.serverURL = "http://" + s.SCEPMapping[CA] + ":8088/scep"
	return nil
}

func (s *deviceService) PostGetCRT(ctx context.Context, keyAlg string, keySize int, c string, st string, l string, o string, ou string, cn string, email string) (data []byte, err error) {
	client := scepclient.NewClient(s.serverURL)

	sigAlgo := x509.SHA1WithRSA

	if client.Supports("SHA-256") || client.Supports("SCEPStandard") {
		if keyAlg == "EC" {
			sigAlgo = x509.ECDSAWithSHA256
		} else {
			sigAlgo = x509.SHA256WithRSA
		}
	}

	var key crypto.PrivateKey
	switch keyAlg {
	case "RSA":
		key, err = newRSAKey(keySize)
	case "EC":
		key, err = newECDSAKey(keySize)
	}
	if err != nil {
		return nil, err
	}
	signerKey, err := loadSignerKey(s.KeyFile)
	if err != nil {
		return nil, errors.New("Unable to load signer key")
	}
	signerCert, err := loadSignerCert(s.CertFile)
	if err != nil {
		return nil, errors.New("Unable to load signer certificate")
	}

	if err != nil {
		return nil, errors.New("Error creating private key")
	}

	opts := &csrOptions{
		cn:       cn,
		org:      o,
		country:  strings.ToUpper(c),
		ou:       ou,
		locality: l,
		province: st,
		key:      key,
		sigAlgo:  sigAlgo,
	}

	csr, err := makeCSR(opts)
	if err != nil {
		return nil, errors.New("Error creating CSR")
	}

	resp, certNum, err := client.GetCACert(ctx)
	if err != nil {
		return nil, err
	}
	var certs []*x509.Certificate
	{
		if certNum > 1 {
			certs, err = scep.CACerts(resp)
			if err != nil {
				return nil, ErrBadRequest
			}
			if len(certs) < 1 {
				return nil, ErrBadRequest
			}
		} else {
			certs, err = x509.ParseCertificates(resp)
			if err != nil {
				return nil, ErrBadRequest
			}
		}
	}

	var msgType scep.MessageType
	{
		msgType = scep.PKCSReq
	}

	recipients := certs

	tmpl := &scep.PKIMessage{
		MessageType: msgType,
		Recipients:  recipients,
		SignerKey:   signerKey,
		SignerCert:  signerCert,
	}

	msg, err := scep.NewCSRRequest(csr, tmpl)
	if err != nil {
		return nil, errors.New("Error in new CSR request")
	}

	var respMsg *scep.PKIMessage

	for {
		// loop in case we get a PENDING response which requires
		// a manual approval.

		respBytes, err := client.PKIOperation(ctx, msg.Raw)
		if err != nil {
			return nil, errors.New("Unable to perform client operation")
		}

		respMsg, err = scep.ParsePKIMessage(respBytes)
		if err != nil {
			return nil, err
		}

		switch respMsg.PKIStatus {
		case scep.FAILURE:
			return nil, ErrBadRequest
		case scep.PENDING:
			time.Sleep(30 * time.Second)
			continue
		}
		break // on scep.SUCCESS
	}

	if err := respMsg.DecryptPKIEnvelope(signerCert, signerKey); err != nil {
		return nil, ErrBadRequest
	}
	respCert := respMsg.CertRepMessage.Certificate
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
	return append(pemCert(respCert.Raw), pemKey(repKey)...), nil
}

type csrOptions struct {
	cn, org, country, ou, locality, province string
	key                                      crypto.PrivateKey
	sigAlgo                                  x509.SignatureAlgorithm
}

func (s *deviceService) authFabricationSystem(authCRT string) error {
	cert, err := certPEM([]byte(authCRT))
	if err != nil {
		return errors.New("Unable to read certificate")
	}
	caCert, err := ioutil.ReadFile(s.CAPath)
	if err != nil {
		return errors.New("Unable to read CA certificate")
	}
	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM([]byte(caCert))
	if !ok {
		return errors.New("Unable to load TLS configuration properties")
	}

	verifyOpts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	if _, err := cert.Verify(verifyOpts); err != nil {
		return errors.New("Unable to verify certificate")
	}
	return nil
}

func loadSignerKey(keyFile string) (*rsa.PrivateKey, error) {
	keyPEM, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	pemBlock, _ := pem.Decode(keyPEM)
	if pemBlock == nil {
		return nil, errors.New("Cannot find the next PEM formatted block")
	}
	if pemBlock.Type != rsaPEMBlockType || len(pemBlock.Headers) != 0 {
		return nil, errors.New("Unmatched type of headers")
	}
	key, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}
	return key, nil

}

func loadSignerCert(certFile string) (*x509.Certificate, error) {
	certPEM, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, err
	}
	pemBlock, _ := pem.Decode(certPEM)
	if pemBlock == nil {
		return nil, errors.New("Cannot find the next PEM formatted block")
	}
	if pemBlock.Type != caPEMBlockType || len(pemBlock.Headers) != 0 {
		return nil, errors.New("Unmatched type of headers")
	}
	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
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

func makeCSR(opts *csrOptions) (*x509.CertificateRequest, error) {
	subject := pkix.Name{
		CommonName:         opts.cn,
		Organization:       subjOrNil(opts.org),
		OrganizationalUnit: subjOrNil(opts.ou),
		Province:           subjOrNil(opts.province),
		Locality:           subjOrNil(opts.locality),
		Country:            subjOrNil(opts.country),
	}
	template := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject:            subject,
			SignatureAlgorithm: opts.sigAlgo,
		},
	}

	derBytes, err := x509util.CreateCertificateRequest(rand.Reader, &template, opts.key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificateRequest(derBytes)
}

// returns nil or []string{input} to populate pkix.Name.Subject
func subjOrNil(input string) []string {
	if input == "" {
		return nil
	}
	return []string{input}
}

const (
	caPEMBlockType  = "CERTIFICATE"
	rsaPEMBlockType = "RSA PRIVATE KEY"
)

func pemKey(derBytes []byte) []byte {
	pemBlock := &pem.Block{
		Type:    privateKeyPEMBlockType,
		Headers: nil,
		Bytes:   derBytes,
	}
	out := pem.EncodeToMemory(pemBlock)
	return out
}

func pemCert(derBytes []byte) []byte {
	pemBlock := &pem.Block{
		Type:    certificatePEMBlockType,
		Headers: nil,
		Bytes:   derBytes,
	}
	out := pem.EncodeToMemory(pemBlock)
	return out
}

func certPEM(data []byte) (*x509.Certificate, error) {
	pemBlock, _ := pem.Decode(data)
	if pemBlock == nil {
		return nil, errors.New("Cannot find the next PEM formatted block")
	}
	if pemBlock.Type != caPEMBlockType || len(pemBlock.Headers) != 0 {
		return nil, errors.New("Unmatched type of headers")
	}
	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}
