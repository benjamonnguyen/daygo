package main

import (
	"fmt"
	"strings"

	"github.com/benjamonnguyen/daygo"
	"github.com/charmbracelet/lipgloss"
)

const (
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorReset  = "\033[0m"
	dash        = '─'
	tailDown    = '┐'
	tailUp      = '┘'
)

var faintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Bold(false)

func line(length int) string {
	var sb strings.Builder
	for range length {
		sb.WriteRune(dash)
	}
	return sb.String()
}

func colorize(color string, s string) string {
	return color + s + colorReset
}

func formatForDisplay(task daygo.ExistingTaskRecord, format string) string {
	return fmt.Sprintf("[%s] %s", task.StartedAt.Format(format), task.Name)
}
