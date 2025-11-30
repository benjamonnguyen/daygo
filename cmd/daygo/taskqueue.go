package main

import (
	"slices"

	"github.com/google/uuid"
)

type TaskQueue interface {
	// Dequeue panics if queue is empty
	Dequeue() Task
	// Peek returns nil if queue is empty
	Peek() *Task
	Queue(t Task)
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

	allTasks            []Task
	filteredTaskIndices []int
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

func (tm *taskQueue) setTasks(tasks []Task) {
	slices.SortFunc(tasks, sortTasks)

	tagToTaskCnt := make(map[string]int)
	for _, task := range tasks {
		for _, tag := range task.Tags {
			tagToTaskCnt[tag] += 1
		}
	}

	tm.allTasks = tasks
	tm.tagToTaskCnt = tagToTaskCnt
	tm.filter()
}

func NewTaskQueue(tasks []Task) TaskQueue {
	tq := taskQueue{}
	tq.setTasks(tasks)
	return &tq
}

func (tm *taskQueue) filter() {
	var filtered []int
	for i, task := range tm.allTasks {
		if tm.filterTag == "" || slices.Contains(task.Tags, tm.filterTag) {
			filtered = append(filtered, i)
		}
	}
	tm.filteredTaskIndices = filtered
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
	tm.setTasks(tm.allTasks)
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

func (tm *taskQueue) AllTags() []string {
	tags := make([]string, 0, len(tm.tagToTaskCnt))
	for tag := range tm.tagToTaskCnt {
		tags = append(tags, tag)
	}
	return tags
}

func (tm *taskQueue) Size() int {
	return len(tm.filteredTaskIndices)
}

func (tm *taskQueue) Queue(t Task) {
	tm.allTasks = append([]Task{t}, tm.allTasks...)
	for _, tag := range t.Tags {
		tm.tagToTaskCnt[tag] += 1
	}
	tm.filter()
}

func (tm *taskQueue) Dequeue() Task {
	task := *tm.Peek()
	i := tm.filteredTaskIndices[len(tm.filteredTaskIndices)-1]
	tm.filteredTaskIndices = tm.filteredTaskIndices[:len(tm.filteredTaskIndices)-1]

	tm.allTasks = slices.Delete(tm.allTasks, i, i+1)

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

	i := tm.filteredTaskIndices[len(tm.filteredTaskIndices)-1]
	return &tm.allTasks[i]
}
