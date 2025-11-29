package main

import (
	"fmt"
	"net/http"
	"os"
	"path"

	txStdLib "github.com/Thiht/transactor/stdlib"

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

	// db
	conn, err := dsdb.Open(dbURL)
	if err != nil {
		panic(err)
	}
	defer conn.Close() //nolint:errcheck
	transactor, dbGetter := txStdLib.NewTransactor(conn.DB(), txStdLib.NestedTransactionsSavepoints)

	// repos
	taskRepo := sqlite.NewTaskRepo(dbGetter, nil)

	// routes
	var c SyncController = &controller{
		transactor: transactor,
		taskRepo:   taskRepo,
	}

	http.HandleFunc("POST /sync", c.Sync)

	// Start the server
	fmt.Printf("Starting sync server on port %s\n", port)
	fmt.Println(http.ListenAndServe(":"+port, nil))
}
