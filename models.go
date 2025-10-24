package daygo

import (
	"time"
)

type Task struct {
	ID          int
	Text        string
	Priority    TaskPriority
	CompletedAt time.Time
}

type TaskPriority int

const (
	PriorityNone TaskPriority = iota
	PriorityThisWeek
	PriorityTomorrow
	PriorityToday
	PriorityASAP
)

type Session struct {
	ID        int
	TaskID    int
	StartedAt time.Time
	EndedAt   time.Time
}
