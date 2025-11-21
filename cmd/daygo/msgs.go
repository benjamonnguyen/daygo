package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type InitMsg struct{}

type EndProgramMsg struct {
	discardPendingTask bool
}

type NewTaskMsg struct {
	task Task
}

type NewNoteMsg struct {
	note Note
}

type QueueTaskMsg struct {
	task string
}

type EditItemMsg struct {
	id   int
	edit string
}

type SkipTaskMsg struct {
	id int
}

type DeletePendingItemMsg struct {
	id int
}

type EndPendingTaskMsg struct {
	id int
}

type TimeBlockMsg struct {
	id       int
	duration time.Duration
}

type FetchTasks struct{}

type AlertMsg struct {
	message string
	color   color
}

func helpAlert(message string) tea.Msg {
	return alert(message, colorYellow)
}

func warningAlert(message string) tea.Msg {
	return alert(message, colorRed)
}

func infoAlert(message string) tea.Msg {
	return alert(message, colorCyan)
}

func alert(message string, color color) tea.Msg {
	return AlertMsg{
		message: message,
		color:   color,
	}
}

type ErrorMsg struct {
	err error
}
