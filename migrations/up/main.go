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
	m, err := migrate.New(
		"file://migrations/",
		os.Getenv("DB_URI"),
	)
	if err != nil {
		log.Fatal(err)
	}

	util.PrintVersion(m)

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("INFO database is up to date")
			os.Exit(0)
		} else {
			log.Fatal(err)
		}
	}

	log.Println("WARN database migrated")

	util.PrintVersion(m)
}
