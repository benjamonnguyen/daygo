package main

import (
	"slices"
	"time"

	"github.com/google/uuid"
)

type TaskQueue interface {
	// Dequeue panics if queue is empty
	Dequeue() Task
	// Peek returns nil if queue is empty
	Peek() *Task
	Queue(t Task) Task
	Size() int
	SetFilter(tag string)
	FilterTag() string
	AllTags() []string

	// sync
	Sync([]Task)
}

type taskQueue struct {
	filterTag    string
	tagToTaskCnt map[string]int

	allTasks      []Task
	filteredTasks []Task
}

func sortTasks(a Task, b Task) int {
	if a.QueuedAt.Before(b.QueuedAt) {
		return 1
	}
	if a.QueuedAt.After(b.QueuedAt) {
		return -1
	}
	return 0
}

func NewTaskQueue(tasks []Task) TaskQueue {
	slices.SortFunc(tasks, sortTasks)

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
	if tm.filterTag == "" {
		tm.filteredTasks = tm.allTasks
	} else {
		var filtered []Task
		for _, task := range tm.allTasks {
			if slices.Contains(task.Tags, tm.filterTag) {
				filtered = append(filtered, task)
			}
		}
		tm.filteredTasks = filtered
	}
}

func (tm *taskQueue) Sync(tasks []Task) {
	if len(tasks) == 0 {
		return
	}
	taskIDToIdx := make(map[uuid.UUID]int)
	for i, t := range tm.allTasks {
		if t.ID != uuid.Nil {
			taskIDToIdx[t.ID] = i
		}
	}
	for _, t := range tasks {
		if i, exists := taskIDToIdx[t.ID]; exists {
			if t.UpdatedAt.After(tm.allTasks[i].UpdatedAt) {
				tm.allTasks[i] = t
			}
		} else {
			tm.allTasks = append(tm.allTasks, t)
		}
	}
	slices.SortFunc(tm.allTasks, sortTasks)
	tm.filter()
}

func (tm *taskQueue) SetFilter(tag string) {
	tm.filterTag = tag
	tm.filter()
}

func (tm *taskQueue) FilterTag() string {
	return tm.filterTag
}

func (tm *taskQueue) CurrentTag() string {
	return tm.filterTag
}

func (tm *taskQueue) Tasks() []Task {
	return tm.filteredTasks
}

func (tm *taskQueue) AllTags() []string {
	tags := make([]string, 0, len(tm.tagToTaskCnt))
	for tag := range tm.tagToTaskCnt {
		tags = append(tags, tag)
	}
	return tags
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
	tm.filteredTasks = tm.filteredTasks[1:]

	tm.allTasks = slices.DeleteFunc(tm.allTasks, func(t Task) bool {
		return t.ID == task.ID
	})

	for _, tag := range task.Tags {
		if tm.tagToTaskCnt[tag] == 1 {
			delete(tm.tagToTaskCnt, tag)
		} else {
			tm.tagToTaskCnt[tag] -= 1
		}
	}

	return task
}

func (tm *taskQueue) Peek() *Task {
	if tm.Size() == 0 {
		return nil
	}

	return &tm.filteredTasks[0]
}
