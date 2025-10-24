package daygo

import (
	"log"
	"os"
	"path"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
}

func LoadConfig() Config {
	cfgDir, _ := os.UserConfigDir()
	cfgDir = path.Join(cfgDir, "daygo")
	confFile := path.Join(cfgDir, "daygo.conf")
	if _, err := os.Stat(confFile); err != nil {
		log.Println("creating default conf file at", confFile)
		if err := os.MkdirAll(cfgDir, 0o744); err != nil {
			panic(err)
		}
		f, err := os.Create(confFile)
		if err != nil {
			panic(err)
		}
		if _, err := f.WriteString("DAYGO_DB_URL=" + path.Join(cfgDir, "daygo.db")); err != nil {
			panic(err)
		}
		_ = f.Close()
	}
	if err := godotenv.Load(confFile); err != nil {
		panic(err)
	}

	return Config{
		DatabaseURL: os.Getenv("DAYGO_DB_URL"),
	}
}
