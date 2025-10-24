package main

import (
	"embed"
	"log"

	"github.com/benjamonnguyen/daygo"
	"github.com/benjamonnguyen/daygo/sqlite"
)

func main() {
	conf := daygo.LoadConfig()

	// db
	var db daygo.Database = sqlite.Open(conf.DatabaseURL)
	//go:embed migrations/*.sql
	var migrations embed.FS
	if err := db.Migrate(migrations); err != nil {
		log.Fatalf("failed migration: %s", err)
	}
	defer func() {
		_ = db.Close()
	}()
}
