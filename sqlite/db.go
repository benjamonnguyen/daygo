// Package sqlite implements daygo's Database interface
package sqlite

import (
	"database/sql"
	"embed"

	"github.com/golang-migrate/migrate/v4"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "modernc.org/sqlite"
)

type database struct {
	conn *sql.DB
}

func Open(url string) *database {
	conn, err := sql.Open("sqlite", url)
	if err != nil {
		panic(err)
	}
	return &database{
		conn: conn,
	}
}

func (db *database) Migrate(migrations embed.FS) error {
	d, err := migratesqlite.WithInstance(db.conn, &migratesqlite.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file:///home/ben/GitHub/daygo/migrations", // TODO dont hard code
		"sqlite", d)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil {
		return err
	}
	return nil
}

func (db *database) Close() error {
	return db.conn.Close()
}
