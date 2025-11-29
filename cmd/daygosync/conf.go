package main

import (
	"os"
	"path"

	"github.com/benjamonnguyen/deadsimple/config"
	"github.com/benjamonnguyen/deadsimple/config/env"
)

const (
	KeyDatabaseURL config.Key = "DAYGO_SYNC_DB_URL"
	KeyPort        config.Key = "DAYGO_SYNC_PORT"
	KeyLogLevel    config.Key = "DAYGO_SYNC_LOG_LEVEL"
	KeyLogPath     config.Key = "DAYGO_SYNC_LOG_PATH"
)

var userHomeDir, _ = os.UserHomeDir()

func LoadConf(src string) (config.Config, error) {
	entries := []env.Entry{
		{
			Key:      KeyDatabaseURL,
			Required: true,
		},
		{
			Key:     KeyPort,
			Default: "8080",
		},
		{
			Key:     KeyLogLevel,
			Default: "INFO",
		},
		{
			Key:     KeyLogPath,
			Default: path.Join(userHomeDir, ".daygo", "sync.log"),
		},
	}

	return env.NewConfig(src, entries...)
}
