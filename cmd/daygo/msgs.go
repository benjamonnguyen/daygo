package main

type InitTaskQueueMsg struct {
	tasks []Task
}

type EndProgramMsg struct {
	discardPendingTask bool
}

type ErrorMsg struct {
	err error
}
