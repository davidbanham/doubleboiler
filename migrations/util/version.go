package util

import (
	"log"

	"github.com/golang-migrate/migrate/v4"
)

func PrintVersion(m *migrate.Migrate) {
	version, dirty, err := m.Version()
	if err == migrate.ErrNilVersion {
		log.Println("INFO database is at version 0")
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("INFO database is at version %d and dirty is %v\n", version, dirty)
}
