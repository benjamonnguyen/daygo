package main

import (
	"io"
	"os"
	"path"

	"github.com/benjamonnguyen/daygo"
	"github.com/benjamonnguyen/deadsimple/config"
	"github.com/benjamonnguyen/deadsimple/config/env"
	"github.com/charmbracelet/log"
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

func Logger(cfg config.Config) daygo.Logger {
	var w io.Writer = os.Stdout
	var logPath, logLvl string
	if err := cfg.GetMany([]config.Key{KeyLogPath, KeyLogLevel}, &logPath, &logLvl); err != nil {
		panic(err)
	}
	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE, 0o644)
		if err != nil {
			panic(err)
		}
		defer f.Close() //nolint:errcheck
		w = f
	}

	lvl, err := log.ParseLevel(logLvl)
	if err != nil {
		panic(err)
	}

	return log.NewWithOptions(w, log.Options{
		Level: lvl,
	})
}
