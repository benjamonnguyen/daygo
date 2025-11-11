package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

type InitMsg struct{}

type EndProgramMsg struct{}

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

type DiscardPendingItemMsg struct {
	id int
}

type AlertMsg struct {
	message string
	color   color
}

func displayHelp(message string) tea.Cmd {
	return displayAlert(message, colorYellow)
}

func displayWarning(message string) tea.Cmd {
	return displayAlert(message, colorRed)
}

func displayAlert(message string, color color) tea.Cmd {
	return func() tea.Msg {
		return AlertMsg{
			message: message,
			color:   color,
		}
	}
}

type ErrorMsg struct {
	err error
}
