package main

import (
	"os"
	"path"

	"github.com/benjamonnguyen/deadsimple/config"
	"github.com/benjamonnguyen/deadsimple/config/env"
)

type Config struct {
	DatabaseURL string
	LogLevel    string
	LogPath     string
	TimeFormat  string
}

const (
	KeyDatabaseURL   config.Key = "DAYGO_DB_URL"
	KeyLogLevel      config.Key = "DAYGO_LOG_LEVEL"
	KeyLogPath       config.Key = "DAYGO_LOG_PATH"
	KeyTimeFormat    config.Key = "DAYGO_TIME_FORMAT"
	KeyDevMode       config.Key = "DAYGO_DEV_MODE"
	KeySyncServerURL config.Key = "DAYGO_SYNC_SERVER_URL"
	KeySyncRate      config.Key = "DAYGO_SYNC_RATE"
)

var (
	userHome, _        = os.UserHomeDir()
	DefaultDatabaseURL = path.Join(userHome, ".daygo", "daygo.db")
	DefaultLogPath     = path.Join(userHome, ".daygo", "daygo.log")
)

func LoadConf(src string) (config.Config, error) {
	entries := []env.Entry{
		{
			Key:      KeyDatabaseURL,
			Default:  DefaultDatabaseURL,
			Required: true,
		},
		{
			Key:     KeyLogLevel,
			Default: "WARN",
		},
		{
			Key:      KeyLogPath,
			Default:  DefaultLogPath,
			Required: true,
		},
		{
			Key:      KeyTimeFormat,
			Default:  "15:04",
			Required: true,
		},
		{
			Key: KeyDevMode,
		},
		{
			Key: KeySyncServerURL,
		},
		{
			Key:      KeySyncRate,
			Default:  "5m",
			Required: true,
		},
	}

	return env.NewConfig(src, entries...)
}
