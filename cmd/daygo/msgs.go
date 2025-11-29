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

type SyncMsg struct {
	tasksToQueue []Task
	error        string
}
