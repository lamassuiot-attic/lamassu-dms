package csr

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

type CSRs struct {
	CSRs []CSR `json:"CSRs"`
}

const (
	PendingStatus  = "NEW"
	ApprobedStatus = "APPROBED"
	DeniedStatus   = "DENIED"
	RevokedStatus  = "REVOKED"
)
