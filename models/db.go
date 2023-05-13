package models

import (
	"database/sql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source"

	_ "github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

func ConnectDatabase(dbName string) error {
	db, err := sql.Open("sqlite", dbName)
	if err != nil {
		return err
	}

	DB = db
	return nil
}

func MigrateDatabase(fs source.Driver) error {
	driver, err := sqlite.WithInstance(DB, &sqlite.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", fs, "sqlite", driver)
	if err != nil {
		return err
	}
	err = m.Up()
	return err
}
