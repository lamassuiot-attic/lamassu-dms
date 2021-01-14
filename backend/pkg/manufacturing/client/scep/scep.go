package scep

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"device-manufacturing-system/pkg/manufacturing/client"
	"device-manufacturing-system/pkg/manufacturing/utils"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	consulsd "github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
	scepclient "github.com/micromdm/scep/client"
	"github.com/micromdm/scep/scep"
)

const (
	caPEMBlockType  = "CERTIFICATE"
	rsaPEMBlockType = "RSA PRIVATE KEY"
)

type SCEP struct {
	keyFile        string
	certFile       string
	proxyAddress   string
	consulProtocol string
	consulHost     string
	consulPort     string
	SCEPMapping    map[string]string
	proxyCA        string
	remote         scepclient.Client
	logger         log.Logger
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

func NewClient(certFile string, keyFile string, proxyAddress string, consulProtocol string, consulHost string, consulPort string, SCEPMapping map[string]string, proxyCA string, logger log.Logger) client.Client {
	return &SCEP{
		certFile:       certFile,
		keyFile:        keyFile,
		proxyAddress:   proxyAddress,
		consulProtocol: consulProtocol,
		consulHost:     consulHost,
		consulPort:     consulPort,
		SCEPMapping:    SCEPMapping,
		proxyCA:        proxyCA,
		logger:         logger,
	}
}

func (s *SCEP) StartRemoteClient(CA string, authCRT []tls.Certificate) error {
	ctx := context.Background()
	serverURL := s.proxyAddress + "/" + s.SCEPMapping[CA] + "/"
	caCertPool, err := utils.CreateCAPool(s.proxyCA)
	if err != nil {
		level.Error(s.logger).Log("err", err, "msg", "Could not create CA Pool to validate SCEP Proxy server")
		return err
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
	consulClient, err := api.NewClient(consulConfig)
	if err != nil {
		level.Error(s.logger).Log("err", err, "msg", "Could not start Consul API Client")
		return ErrConsulConnection
	}
	clientConsul := consulsd.NewClient(consulClient)
	tags := []string{"scep", s.SCEPMapping[CA]}
	passingOnly := true
	duration := 500 * time.Millisecond
	instancer := consulsd.NewInstancer(clientConsul, s.logger, s.SCEPMapping[CA], tags, passingOnly)

	scepClient, err := scepclient.NewSD(serverURL, duration, instancer, s.logger, httpc)
	if err != nil {
		level.Error(s.logger).Log("err", err, "msg", "Could not start SCEP Server Client")
		return err
	}
	s.remote = scepClient
	level.Info(s.logger).Log("msg", "SCEP Server Client started")
	_, err = s.remote.GetCACaps(ctx)
	if err != nil {
		level.Error(s.logger).Log("err", err, "msg", "Could not get CA capabilities from SCEP Server")
		return ErrRemoteConnection
	}
	level.Info(s.logger).Log("msg", "SCEP Server CA capabilities succesfully obtained")
	return nil
}

func (s *SCEP) GetCertificate(keyAlg string, keySize int, c string, st string, l string, o string, ou string, cn string, email string) (*x509.Certificate, crypto.PrivateKey, error) {
	ctx := context.Background()

	sigAlgo := s.checkSignatureAlgorithm(keyAlg)
	level.Info(s.logger).Log("msg", "SCEP Server signature algorithm checked")

	key, err := makeKey(keyAlg, keySize)
	if err != nil {
		level.Error(s.logger).Log("err", err, "msg", "Could not create key for SCEP request")
		return nil, nil, err
	}
	level.Info(s.logger).Log("msg", "Key for SCEP request created")

	sigCert, sigKey, err := loadSignerInfo(s.certFile, s.keyFile)
	if err != nil {
		level.Error(s.logger).Log("err", err, "msg", "Could not load SCEP request signer information")
		return nil, nil, ErrSignerInfoLoading
	}
	level.Info(s.logger).Log("msg", "SCEP request signer information loaded")

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
		return nil, nil, ErrCSRCreate
	}
	level.Info(s.logger).Log("msg", "CSR for SCEP request created")

	resp, certNum, err := s.remote.GetCACert(ctx)
	if err != nil {
		level.Error(s.logger).Log("err", err, "msg", "Could not get CA certificate from SCEP Server")
		return nil, nil, ErrGetRemoteCA
	}

	var certs []*x509.Certificate
	{
		if certNum > 1 {
			certs, err = scep.CACerts(resp)
			if err != nil {
				level.Error(s.logger).Log("err", err, "msg", "Could not get CA certificate from SCEP Server")
				return nil, nil, ErrGetRemoteCA
			}
			if len(certs) < 1 {
				level.Error(s.logger).Log("err", err, "msg", "Could not get CA certificate from SCEP Server")
				return nil, nil, ErrGetRemoteCA
			}
		} else {
			certs, err = x509.ParseCertificates(resp)
			if err != nil {
				level.Error(s.logger).Log("err", err, "msg", "Could not parse CA certificate obtained from SCEP Server")
				return nil, nil, ErrGetRemoteCA
			}
		}
	}
	level.Info(s.logger).Log("msg", "CA certificate obtained from SCEP Server")

	var msgType scep.MessageType
	{
		msgType = scep.PKCSReq
	}

	tmpl := &scep.PKIMessage{
		MessageType: msgType,
		Recipients:  certs,
		SignerKey:   sigKey,
		SignerCert:  sigCert,
	}

	msg, err := scep.NewCSRRequest(csr, tmpl)
	if err != nil {
		level.Error(s.logger).Log("err", err, "msg", "Could not create CSR Request SCEP message")
		return nil, nil, ErrCSRRequestCreate
	}
	level.Info(s.logger).Log("msg", "SCEP CSR Request message created")

	var respMsg *scep.PKIMessage

	for {
		// loop in case we get a PENDING response which requires
		// a manual approval.

		respBytes, err := s.remote.PKIOperation(ctx, msg.Raw)
		if err != nil {
			level.Error(s.logger).Log("err", err, "msg", "Could not perform PKI operation")
			return nil, nil, ErrPKIOperation
		}

		respMsg, err = scep.ParsePKIMessage(respBytes)
		if err != nil {
			level.Error(s.logger).Log("err", err, "msg", "Could not parse PKI message")
			return nil, nil, err
		}

		switch respMsg.PKIStatus {
		case scep.FAILURE:
			err = encodeSCEPFailure(respMsg.FailInfo)
			level.Error(s.logger).Log("err", err, "msg", "PKI operation failed")
			return nil, nil, err
		case scep.PENDING:
			time.Sleep(30 * time.Second)
			continue
		}
		break // on scep.SUCCESS
	}
	level.Info(s.logger).Log("msg", "PKI operation performed successfully")

	if err := respMsg.DecryptPKIEnvelope(sigCert, sigKey); err != nil {
		level.Error(s.logger).Log("err", err, "msg", "Could not decrypt PKI envelope")
		return nil, nil, ErrPKIOperation
	}
	respCert := respMsg.CertRepMessage.Certificate
	level.Info(s.logger).Log("msg", "PKI envelope decrypted and certificate obtained")
	return respCert, key, nil
}

func (s *SCEP) checkSignatureAlgorithm(keyAlg string) x509.SignatureAlgorithm {
	sigAlgo := x509.SHA1WithRSA
	if s.remote.Supports("SHA-256") || s.remote.Supports("SCEPStandard") {
		if keyAlg == "EC" {
			sigAlgo = x509.ECDSAWithSHA256
		} else {
			sigAlgo = x509.SHA256WithRSA
		}
	}
	return sigAlgo
}

func encodeSCEPFailure(fi scep.FailInfo) error {
	switch fi {
	case scep.BadAlg:
		return errors.New("bad algorithm from remote server")
	case scep.BadMessageCheck:
		return errors.New("bad message check from remote server")
	case scep.BadRequest:
		return errors.New("bad request from remote server")
	case scep.BadTime:
		return errors.New("bad time from remote server")
	case scep.BadCertID:
		return errors.New("bad cert ID from remote server")
	default:
		return errors.New("bad request from remote server")
	}
	return nil
}
