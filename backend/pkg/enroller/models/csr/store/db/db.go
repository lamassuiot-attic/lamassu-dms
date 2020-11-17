package db

import (
	"database/sql"
	"device-manufacturing-system/pkg/enroller/models/csr"
	"device-manufacturing-system/pkg/enroller/models/csr/store"
	"fmt"

	_ "github.com/lib/pq"
)

func NewDB(driverName string, dataSourceName string) (store.DB, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	err = checkDBAlive(db)
	for err != nil {
		fmt.Println("Trying to connect to DB")
		err = checkDBAlive(db)
	}

	return &DB{db}, nil
}

type DB struct {
	*sql.DB
}

func checkDBAlive(db *sql.DB) error {
	sqlStatement := `
	SELECT WHERE 1=0`
	_, err := db.Query(sqlStatement)
	return err
}

func (db *DB) SelectAllByCN(cn string) csr.CSRs {
	sqlStatement := `
	SELECT * 
	FROM csr_store
	WHERE commonName = $1;
	`

	rows, err := db.Query(sqlStatement, cn)
	if err != nil {
		return csr.CSRs{CSRs: []csr.CSR{}}
	}
	defer rows.Close()
	csrs := make([]csr.CSR, 0)

	for rows.Next() {
		var c csr.CSR
		err := rows.Scan(&c.Id, &c.CountryName, &c.StateOrProvinceName, &c.LocalityName, &c.OrganizationName, &c.OrganizationalUnitName, &c.CommonName, &c.EmailAddress, &c.Status, &c.CsrFilePath)
		if err != nil {
			return csr.CSRs{CSRs: []csr.CSR{}}
		}
		csrs = append(csrs, c)
	}
	if err = rows.Err(); err != nil {
		return csr.CSRs{CSRs: []csr.CSR{}}
	}
	return csr.CSRs{CSRs: csrs}
}

func (db *DB) SelectByID(id int) (csr.CSR, error) {
	sqlStatement := `
	SELECT *
	FROM csr_store
	WHERE id = $1;
	`
	row := db.QueryRow(sqlStatement, id)
	var c csr.CSR
	err := row.Scan(&c.Id, &c.CountryName, &c.StateOrProvinceName, &c.LocalityName, &c.OrganizationName, &c.OrganizationalUnitName, &c.CommonName, &c.EmailAddress, &c.Status, &c.CsrFilePath)
	if err != nil {
		return csr.CSR{}, err
	}
	return c, nil
}
