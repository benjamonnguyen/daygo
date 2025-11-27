package main

import (
	"net/http"
	"os"
	"path"

	"github.com/benjamonnguyen/daygo/sqlite"
	"github.com/benjamonnguyen/deadsimple/config"
	dsdb "github.com/benjamonnguyen/deadsimple/database/sqlite"
)

func main() {
	// cfg
	confDir, _ := os.UserConfigDir()
	cfg, err := LoadConf(path.Join(confDir, "daygo", "daygosync.conf"))
	if err != nil {
		panic(err)
	}
	var dbURL, port string
	if err := cfg.GetMany([]config.Key{
		KeyDatabaseURL,
		KeyPort,
	}, &dbURL, &port); err != nil {
		panic(err)
	}
	logger := Logger(cfg)

	// db
	db, err := dsdb.Open(dbURL)
	if err != nil {
		panic(err)
	}
	defer db.Close() //nolint:errcheck

	// repos
	taskRepo := sqlite.NewTaskRepo(db.DB(), logger)

	// routes
	var c SyncController = &controller{
		taskRepo: taskRepo,
	}

	http.HandleFunc("POST /sync", c.Sync)

	// Start the server
	logger.Info("Starting sync server on port %s", port)
	logger.Fatal(http.ListenAndServe(":"+port, nil))
}
