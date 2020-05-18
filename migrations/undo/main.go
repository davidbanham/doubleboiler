package main

import (
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
	if err := m.Steps(-1); err != nil {
		log.Fatal(err)
	}
}
