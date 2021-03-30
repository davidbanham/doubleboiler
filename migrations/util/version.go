package util

import (
	"context"
	"doubleboiler/logger"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
)

func PrintVersion(m *migrate.Migrate) {
	version, dirty, err := m.Version()
	if err == migrate.ErrNilVersion {
		logger.Log(context.Background(), logger.Info, "database is at version 0")
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	logger.Log(context.Background(), logger.Info, fmt.Sprintf("database is at version %d and dirty is %v", version, dirty))
}
