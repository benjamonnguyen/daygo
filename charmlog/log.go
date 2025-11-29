// Package charmlog provides an implementation of daygo.Logger using charmbracelet/log
package charmlog

import (
	"io"
	"os"

	"github.com/benjamonnguyen/daygo"
	"github.com/charmbracelet/log"
)

type Options struct {
	Writer io.Writer
	Level  string
}

func NewLogger(opts Options) daygo.Logger {
	var w io.Writer = os.Stdout
	if opts.Writer != nil {
		w = opts.Writer
	}

	lvl, err := log.ParseLevel(opts.Level)
	if err != nil {
		lvl = log.InfoLevel
	}

	return log.NewWithOptions(w, log.Options{
		Level: lvl,
	})
}
