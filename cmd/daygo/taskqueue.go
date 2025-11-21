package main

import (
	"slices"
	"time"
)

type TaskQueue interface {
	// Dequeue panics if queue is empty
	Dequeue() Task
	// Peek returns nil if queue is empty
	Peek() *Task
	Queue(t Task) Task
	Size() int
	SetCurrentTag(tag string)
	Tasks() []Task
}

type taskQueue struct {
	currentTag string

	allTasks      []Task
	filteredTasks []Task
	tagToTaskCnt  map[string]int
}

func NewTaskQueue(tasks []Task) TaskQueue {
	slices.SortFunc[[]Task](tasks, func(a Task, b Task) int {
		if a.QueuedAt.Before(b.QueuedAt) {
			return 1
		}
		if a.QueuedAt.After(b.QueuedAt) {
			return -1
		}
		return 0
	})

	tagToTaskCnt := make(map[string]int)
	for _, task := range tasks {
		for _, tag := range task.Tags {
			tagToTaskCnt[tag] += 1
		}
	}

	return &taskQueue{
		allTasks:      tasks,
		filteredTasks: tasks,
		tagToTaskCnt:  tagToTaskCnt,
	}
}

func (tm *taskQueue) filter() {
	if tm.currentTag == "" {
		tm.filteredTasks = tm.allTasks
	} else {
		var filtered []Task
		for _, task := range tm.allTasks {
			if slices.Contains(task.Tags, tm.currentTag) {
				filtered = append(filtered, task)
			}
		}
		tm.filteredTasks = filtered
	}
}

func (tm *taskQueue) SetCurrentTag(tag string) {
	tm.currentTag = tag
	tm.filter()
}

func (tm *taskQueue) Tasks() []Task {
	return tm.filteredTasks
}

func (tm *taskQueue) Size() int {
	return len(tm.filteredTasks)
}

func (tm *taskQueue) Queue(t Task) Task {
	t.QueuedAt = time.Now()
	tm.allTasks = append([]Task{t}, tm.allTasks...)
	for _, tag := range t.Tags {
		tm.tagToTaskCnt[tag] += 1
	}
	tm.filter()
	return t
}

func (tm *taskQueue) Dequeue() Task {
	task := *tm.Peek()
	tm.filteredTasks = tm.filteredTasks[:tm.Size()-1]

	tm.allTasks = slices.DeleteFunc(tm.allTasks, func(t Task) bool {
		return t.ID == task.ID
	})

	for _, tag := range task.Tags {
		tm.tagToTaskCnt[tag] -= 1
	}

	return task
}

func (tm *taskQueue) Peek() *Task {
	if tm.Size() == 0 {
		return nil
	}

	return &tm.filteredTasks[tm.Size()-1]
}
