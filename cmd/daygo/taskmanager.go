package main

import (
	"context"
)

type TaskQueue interface {
	Peek() Task
	Dequeue() Task
	Queue(t Task)
}

type queue[T any] struct {
	elements []T
	headIdx  int
}

func (q *queue[T]) peek() T {
	return q.elements[q.headIdx]
}

func (q *queue[T]) queue(t T) {
	q.elements = append(q.elements, t)
}

// dequeue will panic if empty
func (q *queue[T]) dequeue() T {
	e := q.elements[q.headIdx]
	q.headIdx += 1
	return e
}

func (q *queue[T]) isEmpty() bool {
	return q.headIdx < len(q.elements)
}

type taskManager struct {
	ActiveTag string

	tasks           []Task
	tagToIndexQueue map[string]*queue[int]
}

func NewTaskQueue(ctx context.Context, tasks []Task) TaskQueue {
	tm := taskManager{}

	tm.tasks = make([]Task, 0, len(tasks))
	for _, task := range tasks {
		tm.Queue(task)
	}

	return &tm
}

func (tm *taskManager) Peek() Task {
	q := tm.tagToIndexQueue[tm.ActiveTag]
	if q.isEmpty() {
		return Task{}
	}
	return tm.tasks[q.peek()]
}

func (tm *taskManager) Queue(t Task) {
	tm.tasks = append(tm.tasks, t)
	i := len(tm.tasks) - 1
	tm.tagToIndexQueue[""].queue(i)
	for _, tag := range t.Tags {
		tm.tagToIndexQueue[tag].queue(i)
	}
}

func (tm *taskManager) Dequeue() Task {
	q := tm.tagToIndexQueue[tm.ActiveTag]
	if q.isEmpty() {
		return Task{}
	}
	return tm.tasks[q.dequeue()]
}

