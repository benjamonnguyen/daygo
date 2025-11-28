package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	txStdLib "github.com/Thiht/transactor/stdlib"
	"github.com/benjamonnguyen/daygo"
	"github.com/benjamonnguyen/daygo/sqlite"
	dsdb "github.com/benjamonnguyen/deadsimple/database/sqlite"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

var logger daygo.Logger

//go:embed migrations/*.sql
var migrations embed.FS

func main() {
	// conf
	conf := LoadConfig()
	f, err := os.OpenFile(conf.LogPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o666)
	if err != nil {
		panic(err)
	}
	defer f.Close() //nolint:errcheck
	logger = configLogger(conf.LogLevel, f)
	logger.Info("loaded config", "config", conf)

	// db
	conn, err := dsdb.Open(conf.DatabaseURL)
	if err != nil {
		logger.Error("failed database open", "error", err)
		os.Exit(1)
	}
	if err := conn.RunMigrations(migrations); err != nil {
		logger.Error("failed migration", "error", err)
		os.Exit(1)
	}
	defer conn.Close() //nolint:errcheck

	_, dbGetter := txStdLib.NewTransactor(conn.DB(), txStdLib.NestedTransactionsSavepoints)

	// repos
	taskRepo := sqlite.NewTaskRepo(dbGetter, logger)

	// svcs
	taskSvc := NewTaskSvc(taskRepo)

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

	userinput := textinput.New()
	userinput.Focus()
	userinput.CharLimit = 280
	userinput.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("221"))

	m := model{
		l:          logger,
		timeFormat: conf.TimeFormat,
		taskSvc:    taskSvc,
		cmdTimeout: 3 * time.Second,
		userinput:  userinput,
		vp:         viewport.New(0, 0),
		taskLog:    opts.tasks,
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		logger.Error(err.Error())
	}
}

func configLogger(level string, w io.Writer) daygo.Logger {
	lvl, err := log.ParseLevel(level)
	if err != nil {
		panic(err)
	}

	return log.NewWithOptions(w, log.Options{
		Level: lvl,
	})
}

type options struct {
	tasks      []Task
	showHelp   bool
	shouldExit bool
}

func parseProgramArgs(ctx context.Context, taskSvc TaskSvc) (options, error) {
	var opts options

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
		_, err := taskSvc.UpsertTask(ctx, t)
		if err != nil {
			return options{}, err
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
