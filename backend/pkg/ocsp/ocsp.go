package ocsp

import (
	"bytes"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/crypto/ocsp"
)

func GetRevocationStatus(subject *x509.Certificate, issuer *x509.Certificate) error {
	if len(subject.OCSPServer) == 0 {
		return errors.New("no OCSP server is attached to the certificate")
	}

	ocspHost := subject.OCSPServer[0]
	u, err := url.Parse(ocspHost)
	if err != nil {
		return errors.New("failed to parse OCSP server host")
	}

	ocspClient := http.Client{
		Timeout: 30 * time.Second,
	}

	ocspReq, err := ocsp.CreateRequest(subject, issuer, &ocsp.RequestOptions{})
	if err != nil {
		return errors.New("failed to compose OCSP request object")
	}
	req, err := http.NewRequest("POST", ocspHost, bytes.NewBuffer(ocspReq))
	req.Header.Add("Content-Type", "application/ocsp-request")
	req.Header.Add("Content-Length", string(len(ocspReq)))
	req.Header.Add("Host", u.Hostname())
	res, err := ocspClient.Do(req)
	if err != nil {
		return errrors.New("failed to get response from OCSP server")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.New("HTTP code is not OK")
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.New("failed to extract OCSP server response")
	}

	ocspRes, err := ocsp.ParseResponse(b, issuer)
	if err != nil {
		return errors.New("failed to parse OCSP server response")
	}

	if ocspRes.Status != ocsp.Good {
		return errors.New("bad revocation status. %v: %v", ocspRes.Status, ocspRes.RevocationReason)
	}
	return nil
}
