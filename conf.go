package daygo

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	LogLevel    string
	LogPath     string
	TimeFormat  string
}

const (
	DefaultLogLevel   = "WARN"
	DefaultTimeFormat = "15:04"
)

var (
	userHome, _        = os.UserHomeDir()
	DefaultDatabaseURL = path.Join(userHome, ".daygo", "daygo.db")
	DefaultLogPath     = path.Join(userHome, ".daygo", "daygo.log")
)

func LoadConfig() Config {
	confFromEnv := Config{
		DatabaseURL: os.Getenv("DAYGO_DB_URL"),
		LogLevel:    os.Getenv("DAYGO_LOG_LEVEL"),
		LogPath:     os.Getenv("DAYGO_LOG_PATH"),
		TimeFormat:  os.Getenv("DAYGO_TIME_FORMAT"),
	}

	if os.Getenv("DAYGO_DEV_MODE") != "" {
		fmt.Println("Dev mode is on!")
		confFromEnv.LogLevel = "DEBUG"
		confFromEnv.DatabaseURL = path.Join(os.TempDir(), "daygo-test.db")
		confFromEnv.LogPath = path.Join(userHome, ".daygo", "dev.log")
		f, err := os.OpenFile(confFromEnv.DatabaseURL, os.O_CREATE|os.O_TRUNC, 0o744)
		if err != nil {
			panic(err)
		}
		_ = f.Close()
	}

	// load file
	cfgDir, _ := os.UserConfigDir()
	cfgDir = path.Join(cfgDir, "daygo")
	confFile := path.Join(cfgDir, "daygo.conf")
	if _, err := os.Stat(confFile); err != nil {
		log.Println("creating default conf file")
		if err := os.MkdirAll(cfgDir, 0o744); err != nil {
			panic(err)
		}
		f, err := os.Create(confFile)
		if err != nil {
			panic(err)
		}
		if _, err := f.WriteString("DAYGO_DB_URL=" + DefaultDatabaseURL); err != nil {
			panic(err)
		}
		if _, err := f.WriteString("DAYGO_LOG_LEVEL=" + DefaultLogLevel); err != nil {
			panic(err)
		}
		if _, err := f.WriteString("DAYGO_LOG_PATH=" + DefaultLogPath); err != nil {
			panic(err)
		}
		if _, err := f.WriteString("DAYGO_TIME_FORMAT=" + DefaultTimeFormat); err != nil {
			panic(err)
		}
		_ = f.Close()
	}
	if err := godotenv.Load(confFile); err != nil {
		panic(err)
	}
	confFromFile := Config{
		DatabaseURL: os.Getenv("DAYGO_DB_URL"),
		LogLevel:    os.Getenv("DAYGO_LOG_LEVEL"),
		LogPath:     os.Getenv("DAYGO_LOG_PATH"),
		TimeFormat:  os.Getenv("DAYGO_TIME_FORMAT"),
	}

	return Config{
		DatabaseURL: coalesce(confFromEnv.DatabaseURL, confFromFile.DatabaseURL, DefaultDatabaseURL),
		LogLevel:    coalesce(confFromEnv.LogLevel, confFromFile.LogLevel, DefaultLogLevel),
		LogPath:     coalesce(confFromEnv.LogPath, confFromFile.LogPath, DefaultLogPath),
		TimeFormat:  coalesce(confFromEnv.TimeFormat, confFromFile.TimeFormat, DefaultTimeFormat),
	}
}

func coalesce(args ...string) string {
	for _, s := range args {
		if s != "" {
			return s
		}
	}
	return ""
}
