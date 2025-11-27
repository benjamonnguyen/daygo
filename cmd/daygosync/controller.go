package main

import (
	"net/http"
	"time"

	"github.com/benjamonnguyen/daygo"
)

type SyncController interface {
	Sync(http.ResponseWriter, *http.Request)
}

type controller struct {
	taskRepo daygo.TaskRepo
}

type SyncRequest struct {
	Since       time.Time
	ClientTasks []daygo.ExistingTaskRecord
}

type SyncResponse struct {
	ServerTasks []daygo.ExistingTaskRecord
}

func (c *controller) Sync(w http.ResponseWriter, r *http.Request) {
}
