package csr

import (
	"encoding/json"
	"errors"
	"fmt"
)

type CSR struct {
	Id                     int    `json:"id"`
	CountryName            string `json:"c"`
	StateOrProvinceName    string `json:"st"`
	LocalityName           string `json:"l"`
	OrganizationName       string `json:"o"`
	OrganizationalUnitName string `json:"ou,omitempty"`
	CommonName             string `json:"cn"`
	EmailAddress           string `json:"mail,omitempty"`
	Status                 string `json:"status"`
	CsrFilePath            string `json:"csrpath,omitempty"`
}

type EmbeddedCSRs struct {
	CSRs CSR `json:"csr"`
}

type CSRs struct {
	CSRs []CSR `json:"csr"`
}

type Data struct {
	*EmbeddedCSRs
	*CSRs
}

func (d *Data) UnmarshalJSON(data []byte) error {
	temp := struct {
		Embedded interface{} `json:"csr"`
	}{}
	err := json.Unmarshal(data, &temp)
	if err != nil {
		return err
	}
	switch temp.Embedded.(type) {
	case map[string]interface{}:
		var c EmbeddedCSRs
		if err := json.Unmarshal(data, &c); err != nil {
			return err
		}
		d.EmbeddedCSRs = &c
		d.CSRs = nil
		fmt.Printf("%v\n", c)
	case []interface{}:
		var c CSRs
		if err := json.Unmarshal(data, &c); err != nil {
			return err
		}
		d.CSRs = &c
		d.EmbeddedCSRs = nil
	default:
		return errors.New("Invalid object type")
	}
	return nil
}

const (
	PendingStatus  = "NEW"
	ApprobedStatus = "APPROBED"
	DeniedStatus   = "DENIED"
	RevokedStatus  = "REVOKED"
)
