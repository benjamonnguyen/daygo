package main

import (
	"fmt"
)

type InitMsg struct{}

type NewTaskMsg struct {
	task Task
}

type NewNoteMsg struct {
	note Note
}

type QueueMsg struct {
	task string
}

type SkipMsg struct {
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
