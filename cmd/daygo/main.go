package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	txStdLib "github.com/Thiht/transactor/stdlib"
	"github.com/benjamonnguyen/daygo"
	"github.com/benjamonnguyen/daygo/charmlog"
	"github.com/benjamonnguyen/daygo/sqlite"
	"github.com/benjamonnguyen/deadsimple/config"
	dsdb "github.com/benjamonnguyen/deadsimple/database/sqlite"
	tea "github.com/charmbracelet/bubbletea"
)

//go:embed migrations/*.sql
var migrations embed.FS

var logger daygo.Logger

func main() {
	// cfg
	confDir, _ := os.UserConfigDir()
	cfg, err := LoadConf(path.Join(confDir, "daygo", "daygo.conf"))
	if err != nil {
		panic(err)
	}
	var logPath, logLvl, dbURL, timeFormat, syncServerURL, syncRate string
	if err := cfg.GetMany([]config.Key{
		KeyLogPath,
		KeyLogLevel,
		KeyDatabaseURL,
		KeyTimeFormat,
		KeySyncServerURL,
		KeySyncRate,
	}, &logPath, &logLvl, &dbURL, &timeFormat, &syncServerURL, &syncRate); err != nil {
		panic(err)
	}
	sr, err := time.ParseDuration(syncRate)
	if err != nil {
		panic(err)
	}

	// logger
	var w io.Writer
	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE, 0o644)
		if err != nil {
			panic(err)
		}
		defer f.Close() //nolint:errcheck
	}
	logger = charmlog.NewLogger(charmlog.Options{
		Writer: w,
		Level:  logLvl,
	})
	logger.Info("loaded config", "config", cfg)

	// db
	conn, err := dsdb.Open(dbURL)
	if err != nil {
		logger.Error("failed database open", "error", err)
		os.Exit(1)
	}
	if err := conn.RunMigrations(migrations); err != nil {
		logger.Error("failed migration", "error", err)
		os.Exit(1)
	}
	defer conn.Close() //nolint:errcheck

	transactor, dbGetter := txStdLib.NewTransactor(conn.DB(), txStdLib.NestedTransactionsSavepoints)

	// repos
	taskRepo := sqlite.NewTaskRepo(dbGetter, logger)
	syncSessionRepo := sqlite.NewSyncSessionRepo(dbGetter, logger)

	// svcs
	taskSvc := NewTaskSvc(transactor, logger, taskRepo, syncSessionRepo)

	// handle initial args
	timeout, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	opts, err := parseProgramArgs(timeout, taskSvc)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if opts.showHelp {
		fmt.Println(colorize(colorYellow, programUsage))
		os.Exit(0)
	}
	if opts.shouldExit {
		os.Exit(0)
	}

	// start program
	fmt.Println(colorize(colorYellow, logo))
	fmt.Printf("\nEnter \"/h\" for help\n\n")

	m := NewModel(taskSvc, opts.tasks, logger, modelOptions{
		cmdTimeout:    3 * time.Second,
		timeFormat:    timeFormat,
		syncServerURL: syncServerURL,
		syncRate:      sr,
	})
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		logger.Error(err.Error())
	}
}

type programOptions struct {
	tasks      []Task
	showHelp   bool
	shouldExit bool
}

func parseProgramArgs(ctx context.Context, taskSvc TaskSvc) (programOptions, error) {
	var opts programOptions

	if len(os.Args) == 1 {
		return opts, nil
	}

	var cmd, arg string
	if strings.HasPrefix(os.Args[1], "/") {
		cmd = os.Args[1]
		if len(os.Args) > 2 {
			arg = strings.Join(os.Args[2:], " ")
		}
	} else {
		arg = strings.Join(os.Args[1:], " ")
	}

	logger.Debug("parsed program args", "cmd", cmd, "arg", arg)
	switch cmd {
	case "/n", "":
		if arg != "" {
			t := TaskFromName(arg)
			t.StartedAt = time.Now()
			opts.tasks = append(opts.tasks, t)
			return opts, nil
		}

		return opts, nil
	case "/a":
		t := TaskFromName(arg)
		t.UpdatedAt = time.Now()
		_, err := taskSvc.UpsertTask(ctx, t)
		if err != nil {
			return programOptions{}, err
		}
		fmt.Printf(`Queued up "%s"`+"\n", arg)
		opts.shouldExit = true
		return opts, nil
	case "/review":
		// TODO /review
		panic("review command not implemented")
	default:
		opts.showHelp = true
		return opts, nil
	}
}
