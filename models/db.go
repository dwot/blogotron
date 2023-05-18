package models

import (
	"database/sql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source"
	"golang/util"
	"strconv"

	_ "github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"
)

var DB *sql.DB
var targetVersion = 2

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
	mv, _, _ := m.Version()
	util.Logger.Info().Msg("Current DB Version:" + strconv.FormatUint(uint64(mv), 10))
	curVer := int(mv)
	if curVer < targetVersion {
		err = m.Up()
		return err
	} else {
		util.Logger.Info().Msg("At expected version: " + strconv.Itoa(targetVersion) + " skipping migration.")
	}
	return err
}
