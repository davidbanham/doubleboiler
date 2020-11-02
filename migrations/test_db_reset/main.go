package main

import (
	"doubleboiler/migrations/util"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if os.Getenv("STAGE") != "testing" {
		log.Fatal("Reset may only be run against testing")
	}

	m, err := migrate.New(
		"file://migrations/",
		os.Getenv("DB_URI"),
	)
	if err != nil {
		log.Fatal(err)
	}

	util.PrintVersion(m)

	if err := m.Down(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("INFO database is at version 0")
		} else {
			log.Fatal("ERROR migrating down", err)
		}
	}

	util.PrintVersion(m)

	if err := m.Up(); err != nil {
		log.Fatal("ERROR migrating up", err)
	}

	util.PrintVersion(m)

}
