package utils

import (
	"encoding/pem"
	"errors"
)

const (
	CertPEMBlockType = "CERTIFICATE"
	KeyPEMBlockType  = "RSA PRIVATE KEY"
)

func PEMKey(derBytes []byte) []byte {
	pemBlock := &pem.Block{
		Type:    KeyPEMBlockType,
		Headers: nil,
		Bytes:   derBytes,
	}
	out := pem.EncodeToMemory(pemBlock)
	return out
}

func PEMCert(derBytes []byte) []byte {
	pemBlock := &pem.Block{
		Type:    CertPEMBlockType,
		Headers: nil,
		Bytes:   derBytes,
	}
	out := pem.EncodeToMemory(pemBlock)
	return out
}

func CheckPEMBlock(pemBlock *pem.Block, blockType string) error {
	if pemBlock == nil {
		return errors.New("cannot find the next PEM formatted block")
	}
	if pemBlock.Type != blockType || len(pemBlock.Headers) != 0 {
		return errors.New("unmatched type of headers")
	}
	return nil
}
