package store

import "device-manufacturing-system/pkg/enroller/models/csr"

type DB interface {
	SelectAllByCN(cn string) csr.CSRs
	SelectByID(id int) (csr.CSR, error)
}
