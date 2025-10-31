package main

import (
	"embed"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/benjamonnguyen/daygo"
	"github.com/benjamonnguyen/daygo/sqlite"
	dsdb "github.com/benjamonnguyen/deadsimple/database/sqlite"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var logger *slog.Logger

//go:embed migrations/*.sql
var migrations embed.FS

func main() {
	// conf
	conf := daygo.LoadConfig()
	f, err := os.OpenFile(conf.LogPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	logger = configLogger(conf.LogLevel, f)
	logger.Info("loaded config", "config", conf)

	// db
	db, err := dsdb.Open(conf.DatabaseURL)
	if err != nil {
		logger.Error("failed database open", "error", err)
		os.Exit(1)
	}
	if err := db.RunMigrations(migrations); err != nil {
		logger.Error("failed migration", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = db.Close()
	}()

	// repos
	taskRepo := sqlite.NewTaskRepo(db.Conn(), logger)

	// svcs
	taskSvc := NewTaskSvc(taskRepo)

	// start program
	m := model{
		l:          logger,
		timeFormat: conf.TimeFormat,
		taskSvc:    taskSvc,
		cmdTimeout: 3 * time.Second,
	}
	cmd, err := m.parseProgramArgs()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	msg := cmd()
	// TODO jank, parseProgramArgs should be decoupled from model
	if msg, ok := msg.(NewTaskMsg); ok {
		m.initialMsg = msg
	}

	fmt.Println(colorize(colorYellow, logo))
	fmt.Printf("\nEnter \"/h\" for help\n\n")

	userinput := textinput.New()
	userinput.Focus()
	userinput.CharLimit = 280
	userinput.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("221"))

	m.userinput = userinput
	m.vp = viewport.New(0, 0)

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		logger.Error(err.Error())
	}
}

func configLogger(level string, w io.Writer) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "DEBUG":
		lvl = slog.LevelDebug
	case "INFO":
		lvl = slog.LevelInfo
	case "WARN":
		lvl = slog.LevelWarn
	case "ERROR":
		lvl = slog.LevelError
	}

	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level:     lvl,
		AddSource: true,
	})

	return slog.New(handler)
}
