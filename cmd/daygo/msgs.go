package main

type InitTaskQueueMsg struct {
	tasks []Task
}

type EndProgramMsg struct {
	discardPendingTask bool
}

type ErrorMsg struct {
	err     error
	isFatal bool
}

type SyncMsg struct {
	tasksToQueue      []Task
	toServerSyncCount int
	err               string
}

type QueueMsg struct {
	task Task
}

type QueueMsg struct {
	task Task
}
