package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

type InitTaskQueueMsg struct {
	tasks []Task
}

type EndProgramMsg struct {
	discardPendingTask bool
}

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
