package main

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/timer"
)

type timeBlockTimer struct {
	timer.Model
}

func (t timeBlockTimer) View() string {
	dur := ""
	if t.Timeout > time.Minute {
		dur = fmt.Sprintf("%dm", int(t.Timeout.Minutes()))
	} else {
		dur = fmt.Sprintf("%ds", int(t.Timeout.Seconds()))
	}
	return fmt.Sprintf("Task time blocked for %s", dur)
}
