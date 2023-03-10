package sqlite

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/elementsproject/peerswap/log"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite" // Register relevant drivers.
)

//go:embed migrations/*.sql
var fs embed.FS

func Migrate(db *sql.DB) (uint, error) {
	log.Infof("Checking sqlite version for migrations.")
	d, err := iofs.New(fs, "migrations")
	if err != nil {
		return 0, fmt.Errorf("failed to read migration files: %w", err)
	}
	instance, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return 0, fmt.Errorf("failed to create database instance: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", d, "", instance)
	if err != nil {
		return 0, fmt.Errorf("failed to create database instance: %w", err)
	}

	oldversion, dirty, err := m.Version()
	switch {
	case err == migrate.ErrNilVersion:
		log.Infof("Sqlite database not yet created, creating database.")

	case err != nil:
		return 0, fmt.Errorf("cannot retrieve version: %w", err)

	case dirty:
		log.Infof("Sqlite database migration is dirty, attempting to force to previous version.")
		forceVersion := oldversion - 1
		err := m.Force(int(forceVersion))
		if err != nil {
			return 0, fmt.Errorf("cannot force version %v: %w",
				forceVersion, err)
		}
	}

	err = m.Up()
	switch {
	case err == migrate.ErrNoChange:
		log.Infof("No database migrations required.")
	case err != nil:
		return 0, fmt.Errorf("failed to migrate database: %w", err)
	}

	newversion, _, err := m.Version()
	if err != nil {
		return 0, fmt.Errorf("cannot retrieve version after migration: %w", err)
	}

	log.Infof("Successfully migrated database from version '%v' to version '%v'", oldversion, newversion)
	return newversion, nil
}
