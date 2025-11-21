package main

import (
	"fmt"
	"strings"

	"github.com/benjamonnguyen/daygo"
)

// models

type Task struct {
	daygo.ExistingTaskRecord
	Notes      []Note
	IsTerminal bool
}

type Note daygo.TaskRecord

func (t *Task) IsPending() bool {
	return t != nil && !t.StartedAt.IsZero() && t.EndedAt.IsZero()
}

func (t Task) LastNote() *Note {
	if len(t.Notes) > 0 {
		return &t.Notes[len(t.Notes)-1]
	}
	return nil
}

func (t Task) Render(timeFormat string) (string, int) {
	const minLineWidth = 20
	maxItemWidth := len(t.Name)
	var notes []string
	for _, note := range t.Notes {
		if len(note.Name) > maxItemWidth {
			maxItemWidth = len(note.Name)
		}
		notes = append(notes, note.Render(timeFormat))
	}

	l := maxItemWidth + 10
	l = max(minLineWidth, l)

	forDisplay := formatForDisplay(t.TaskRecord, timeFormat)
	taskLine := fmt.Sprintf("%s %s%c", forDisplay, line(l-len(forDisplay)), tailDown)
	lines := []string{
		taskLine,
	}
	lines = append(lines, notes...)
	if t.IsTerminal {
		endTime := t.EndedAt.Format(timeFormat)
		lines = append(lines, fmt.Sprintf(
			"[%s] %s%c",
			endTime,
			line(l-len(endTime)-2),
			tailUp,
		))
	} else if !t.IsPending() {
		lines = append(lines, fmt.Sprintf("%s%c", line(l+1), tailUp))
	}

	return strings.Join(lines, "\n"), len(lines)
}

func (n Note) Render(timeFormat string) string {
	return formatForDisplay(daygo.TaskRecord(n), timeFormat)
}
