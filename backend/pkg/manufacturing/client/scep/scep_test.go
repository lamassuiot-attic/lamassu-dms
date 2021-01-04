package scep

import (
	"crypto/tls"
	"device-manufacturing-system/pkg/manufacturing/configs"
	"device-manufacturing-system/pkg/manufacturing/utils"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/go-kit/kit/log"
	scepclient "github.com/micromdm/scep/client"
)

func TestStartRemoteClient(t *testing.T) {
	scep := setup(t)

	validcert, err := tls.LoadX509KeyPair("testdata/valid.crt", "testdata/valid.key")
	if err != nil {
		t.Fatal("Unable to read valid certificate and key")
	}

	revokedcert, err := tls.LoadX509KeyPair("testdata/revoked.crt", "testdata/valid.key")
	if err != nil {
		t.Fatal("Unable to read revoked certificate and key")
	}

	selfcert, err := tls.LoadX509KeyPair("testdata/self.crt", "testdata/self.key")
	if err != nil {
		t.Fatal("Unable to read self-signed certificate and key")
	}

	testCases := []struct {
		name    string
		ca      string
		authCRT []tls.Certificate
		ret     error
	}{
		{"Incorrect CA", "doesNotExist", []tls.Certificate{validcert}, ErrRemoteConnection},
		{"Self-signed authentication certificate", "Lamassu-Root-CA1-RSA4096", []tls.Certificate{selfcert}, ErrRemoteConnection},
		{"Revoked authentication certificate", "Lamassu-Root-CA1-RSA4096", []tls.Certificate{revokedcert}, ErrRemoteConnection},
		{"Correct CA and authentication certificate", "Lamassu-Root-CA1-RSA4096", []tls.Certificate{validcert}, nil},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing %s", tc.name), func(t *testing.T) {
			err := scep.StartRemoteClient(tc.ca, tc.authCRT)
			if tc.ret != err {
				t.Errorf("Got result is %s; want %s", err, tc.ret)
			}
		})
	}
}

func TestGetCertificate(t *testing.T) {
	scep := setup(t)
	serverURL := scep.proxyAddress + "/" + scep.SCEPMapping["Lamassu-Root-CA1-RSA4096"] + "/"
	caCertPool, err := utils.CreateCAPool(scep.proxyCA)
	if err != nil {
		t.Fatal("Unable to create proxy CA pool")
	}

	validcert, err := tls.LoadX509KeyPair("testdata/valid.crt", "testdata/valid.key")
	if err != nil {
		t.Fatal("Unable to read valid certificate and key")
	}

	httpc := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{validcert},
			},
		},
	}
	scep.remote, err = scepclient.New(serverURL, scep.logger, httpc)

	crt, _, err := scep.GetCertificate("RSA", 2048, "test", "test", "test", "test", "test", "test.com", "")

	if err != nil {
		t.Errorf("Error obtaining certificate: %s", err.Error())
	} else {
		if crt.Subject.CommonName != "test.com" {
			t.Error("Certificate common name does not match with request common name")
		}

		if crt.Issuer.CommonName != "LKS Next Root CA 1" {
			t.Errorf("Certificate has not been issued by correct CA, issuer is: %s", crt.Issuer.CommonName)
		}
	}

}

func setup(t *testing.T) *SCEP {
	t.Helper()

	cfg, err := configs.NewConfig("manufacturingtest")
	if err != nil {
		t.Fatal("Unable to get configuration variables")
	}

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}

	return &SCEP{
		certFile:     cfg.CertFile,
		keyFile:      cfg.KeyFile,
		proxyAddress: cfg.ProxyAddress,
		SCEPMapping:  cfg.SCEPMapping,
		proxyCA:      cfg.ProxyCA,
		logger:       logger}
}
