package extension

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/lamassuiot/device-manufacturing-system/pkg/manufacturing/client"
	"github.com/lamassuiot/device-manufacturing-system/pkg/manufacturing/utils"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	consulsd "github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
	extensionclient "github.com/micromdm/scep/client/extension"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/lamassuiot/lamassu-est/client/estclient"
)

const (
	caPEMBlockType  = "CERTIFICATE"
	rsaPEMBlockType = "RSA PRIVATE KEY"
)

type SCEPExt struct {
	proxyAddress   string
	consulProtocol string
	consulHost     string
	consulPort     string
	consulCA       string
	proxyCA        string
	extClient      extensionclient.Client
	logger         log.Logger
	otTracer       stdopentracing.Tracer
}

type CSROptions struct {
	cn, org, country, ou, locality, province string
	key                                      crypto.PrivateKey
	sigAlgo                                  x509.SignatureAlgorithm
}

var (
	ErrSignerInfoLoading = errors.New("unable to read Signer info")
	ErrCSRCreate         = errors.New("unable to create CSR")
	ErrPKIOperation      = errors.New("unable to perform PKI operation")
	ErrCSRRequestCreate  = errors.New("unable to create CSR request")
	ErrGetRemoteCA       = errors.New("error getting remote CA certificate")
	ErrRemoteConnection  = errors.New("error connecting to remote server")
	ErrConsulConnection  = errors.New("error connecting to Service Discovery server")
)

func NewClient(proxyAddress string, consulProtocol string, consulHost string, consulPort string, consulCA string, proxyCA string, logger log.Logger, otTracer stdopentracing.Tracer) client.Client {
	return &SCEPExt{
		proxyAddress:   proxyAddress,
		consulProtocol: consulProtocol,
		consulHost:     consulHost,
		consulPort:     consulPort,
		consulCA:       consulCA,
		proxyCA:        proxyCA,
		logger:         logger,
		otTracer:       otTracer,
	}
}

func (s *SCEPExt) createClient(authCRT []tls.Certificate) (extensionclient.Client, error) {
	caCertPool, err := utils.CreateCAPool(s.proxyCA)
	if err != nil {
		level.Error(s.logger).Log("err", err, "msg", "Could not create CA Pool to validate SCEP Extension")
		return nil, err
	}

	httpc := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				Certificates: authCRT,
			},
		},
	}

	consulConfig := api.DefaultConfig()
	consulConfig.Address = s.consulProtocol + "://" + s.consulHost + ":" + s.consulPort
	tlsConf := &api.TLSConfig{CAFile: s.consulCA}
	consulConfig.TLSConfig = *tlsConf
	consulClient, err := api.NewClient(consulConfig)
	if err != nil {
		level.Error(s.logger).Log("err", err, "msg", "Could not start Consul API Client")
		return nil, ErrConsulConnection
	}
	clientConsul := consulsd.NewClient(consulClient)
	tags := []string{"scep", "extension"}
	passingOnly := true
	duration := 500 * time.Millisecond
	instancer := consulsd.NewInstancer(clientConsul, s.logger, "scepextension", tags, passingOnly)

	extClient, err := extensionclient.NewSD(s.proxyAddress, duration, instancer, s.logger, httpc, s.otTracer)
	if err != nil {
		level.Error(s.logger).Log("err", err, "msg", "Could not start SCEP Extension Client")
		return nil, err
	}
	level.Info(s.logger).Log("msg", "SCEP Extension Client started")
	return extClient, nil
}

func (s *SCEPExt) StartClient(ctx context.Context, CA string, authCRT []tls.Certificate) error {
	if s.extClient == nil {
		extClient, err := s.createClient(authCRT)
		if err != nil {
			return err
		}
		s.extClient = extClient
	}
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Second*10))
	defer cancel()
	err := s.extClient.PostSetConfig(ctx, CA)
	if err != nil {
		level.Error(s.logger).Log("err", err, "msg", "Could not set configuration for SCEP Extension")
		return ErrRemoteConnection
	}
	level.Info(s.logger).Log("msg", "SCEP Extension configuration succesfully assigned")
	return nil
}

func (s *SCEPExt) GetCertificate(ctx context.Context, keyAlg string, keySize int, c string, st string, l string, o string, ou string, cn string, email string, caName string) (*x509.Certificate, crypto.PrivateKey, error) {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Second*10))
	defer cancel()
	sigAlgo := s.checkSignatureAlgorithm(keyAlg)
	level.Info(s.logger).Log("msg", "CSR signature algorithm checked")

	key, err := makeKey(keyAlg, keySize)
	if err != nil {
		level.Error(s.logger).Log("err", err, "msg", "Could not create key for SCEP request")
		return nil, nil, err
	}
	level.Info(s.logger).Log("msg", "Key for SCEP request created")

	opts := &CSROptions{
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
		level.Error(s.logger).Log("err", err, "msg", "Could not create CSR for SCEP request")
		return nil, nil, err
	}
	level.Info(s.logger).Log("msg", "CSR for SCEP request created")

	/* SWITCH FROM SCEP TO EST HERE */

	//pemcsr := utils.PEMCSR(csr.Raw)
	//crtData, err := s.extClient.PostGetCRT(ctx, pemcsr)
	crt, err := estclient.Enroll(csr, caName)

	if err != nil {
		level.Error(s.logger).Log("err", err, "msg", "Could not obtain certificate from SCEP Server")
		return nil, nil, err
	}
	level.Info(s.logger).Log("msg", "SCEP Server returned certificate")
	/*crt, err := utils.ParseCertificate(crtData)
	if err != nil {
		level.Error(s.logger).Log("err", err, "msg", "Could not parse certificate obtained from SCEP Server")
		return nil, nil, err
	}*/
	level.Info(s.logger).Log("msg", "Certificate obtained from SCEP Server parsed")
	return crt, key, nil

}

func (s *SCEPExt) checkSignatureAlgorithm(keyAlg string) x509.SignatureAlgorithm {
	sigAlgo := x509.SHA1WithRSA
	if keyAlg == "EC" {
		sigAlgo = x509.ECDSAWithSHA256
	} else {
		sigAlgo = x509.SHA256WithRSA
	}
	return sigAlgo
}
