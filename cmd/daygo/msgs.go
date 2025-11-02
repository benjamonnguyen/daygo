package main

import (
	"fmt"
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

type ErrorMsg struct {
	err error
}

func errorMsg(format string, args ...any) ErrorMsg {
	return ErrorMsg{
		err: fmt.Errorf(format, args...),
	}
}
