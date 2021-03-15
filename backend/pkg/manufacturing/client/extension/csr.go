package extension

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"device-manufacturing-system/crypto/x509util"
)

func makeCSR(opts *CSROptions) (*x509.CertificateRequest, error) {
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
