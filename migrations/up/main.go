package main

import (
	"context"
	"doubleboiler/logger"
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
			logger.Log(context.Background(), logger.Info, "database is up to date")
			os.Exit(0)
		} else {
			log.Fatal(err)
		}
	}

	logger.Log(context.Background(), logger.Warning, "database migrated")

	util.PrintVersion(m)
}
