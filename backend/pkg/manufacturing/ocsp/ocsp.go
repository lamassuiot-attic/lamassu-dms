package ocsp

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/crypto/ocsp"
)

const (
	OCSPSuccess               OCSPStatus = 0
	OCSPNoServer              OCSPStatus = -1
	OCSPFailedParseOCSPHost   OCSPStatus = -2
	OCSPFailedComposeRequest  OCSPStatus = -3
	OCSPFailedSubmit          OCSPStatus = -4
	OCSPFailedResponse        OCSPStatus = -5
	OCSPFailedExtractResponse OCSPStatus = -6
	OCSPFailedParseResponse   OCSPStatus = -7
	OCSPInvalidValidity       OCSPStatus = -8
	OCSPRevokedOrUnknown      OCSPStatus = -9
)

type OCSPStatus int

func GetRevocationStatus(wg *sync.WaitGroup, ocspStatusChan chan<- OCSPStatus, ocspErrorChan chan<- error, subject *x509.Certificate, issuer *x509.Certificate) {
	defer wg.Done()
	if len(subject.OCSPServer) == 0 {
		ocspErrorChan <- fmt.Errorf("no OCSP server is attached to the certificate. %v", subject.Subject)
		ocspStatusChan <- OCSPNoServer
		return
	}

	ocspHost := subject.OCSPServer[0]
	u, err := url.Parse(ocspHost)
	if err != nil {
		ocspErrorChan <- fmt.Errorf("failed to parse OCSP server host. %v", ocspHost)
		ocspStatusChan <- OCSPFailedParseOCSPHost
	}

	ocspClient := http.Client{
		Timeout: 30 * time.Second,
	}

	ocspReq, err := ocsp.CreateRequest(subject, issuer, &ocsp.RequestOptions{})
	if err != nil {
		ocspErrorChan <- fmt.Errorf("failed to compose OCSP request object. %v", subject.Subject)
		ocspStatusChan <- OCSPFailedComposeRequest
	}
	req, err := http.NewRequest("POST", ocspHost, bytes.NewBuffer(ocspReq))
	req.Header.Add("Content-Type", "application/ocsp-request")
	req.Header.Add("Content-Length", string(len(ocspReq)))
	req.Header.Add("Host", u.Hostname())
	res, err := ocspClient.Do(req)
	if err != nil {
		ocspErrorChan <- err
		ocspStatusChan <- OCSPFailedSubmit
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		ocspErrorChan <- fmt.Errorf("HTTP code is not OK. %v: %v", res.StatusCode, res.Status)
		ocspStatusChan <- OCSPFailedResponse
		return
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		ocspErrorChan <- err
		ocspStatusChan <- OCSPFailedExtractResponse
		return
	}

	ocspRes, err := ocsp.ParseResponse(b, issuer)
	if err != nil {
		ocspErrorChan <- err
		ocspStatusChan <- OCSPFailedParseResponse
		return
	}

	if ocspRes.Status != ocsp.Good {
		ocspErrorChan <- fmt.Errorf("bad revocation status. %v: %v, cert: %v", ocspRes.Status, ocspRes.RevocationReason, subject.Subject)
		ocspStatusChan <- OCSPFailedParseResponse
		return
	}
	return
}
