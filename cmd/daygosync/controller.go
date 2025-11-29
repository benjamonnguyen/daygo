package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Thiht/transactor"
	"github.com/benjamonnguyen/daygo"
)

type SyncController interface {
	Sync(http.ResponseWriter, *http.Request)
}

type controller struct {
	transactor transactor.Transactor
	taskRepo   daygo.TaskRepo
}

type SyncRequest struct {
	LastSyncedAt time.Time                  `json:"last_synced_at"`
	ClientTasks  []daygo.ExistingTaskRecord `json:"client_tasks"`
}

type SyncResponse struct {
	ServerTasks []daygo.ExistingTaskRecord `json:"server_tasks"`
}

type httpError struct {
	code int
	msg  string
}

func (err httpError) Error() string {
	return fmt.Sprintf("%s[%d]", err.msg, err.code)
}

func (c *controller) Sync(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Unmarshal body to SyncRequest
	var syncReq SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&syncReq); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if syncReq.LastSyncedAt.IsZero() {
		http.Error(w, "LastSyncedAt field is required", http.StatusBadRequest)
		return
	}

	// Process client tasks with conflict resolution within transaction
	err := c.transactor.WithinTransaction(r.Context(), c.syncClientTasks(r.Context(), syncReq.ClientTasks))
	if err != nil {
		var httpErr httpError
		if errors.As(err, &httpErr) {
			http.Error(w, "Failed to sync client tasks: "+httpErr.msg, httpErr.code)
		} else {
			http.Error(w, "Failed to sync client tasks: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return server tasks to client
	serverTasks, err := c.taskRepo.GetByCreatedTime(r.Context(), syncReq.LastSyncedAt, time.Time{})
	if err != nil {
		http.Error(w, "Failed to get server tasks: "+err.Error(), http.StatusInternalServerError)
		return
	}
	response := SyncResponse{
		ServerTasks: serverTasks,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (c *controller) syncClientTasks(ctx context.Context, tasks []daygo.ExistingTaskRecord) func(context.Context) error {
	return func(context.Context) error {
		var existingTaskIDs []any
		for _, clientTask := range tasks {
			if clientTask.ID > 0 {
				existingTaskIDs = append(existingTaskIDs, clientTask.ID)
			}
		}

		existingTasks, err := c.taskRepo.GetTasks(ctx, existingTaskIDs)
		if err != nil {
			return httpError{
				code: http.StatusInternalServerError,
				msg:  "Failed to get existing tasks: " + err.Error(),
			}
		}

		taskIDToExistingRecord := make(map[int]daygo.ExistingTaskRecord)
		for _, task := range existingTasks {
			taskIDToExistingRecord[task.ID] = task
		}

		for _, clientTask := range tasks {
			serverTask, exists := taskIDToExistingRecord[clientTask.ID]
			if clientTask.ID == 0 || !exists {
				// New task - create it
				_, err := c.taskRepo.InsertTask(ctx, clientTask.TaskRecord)
				if err != nil {
					return httpError{
						code: http.StatusInternalServerError,
						msg:  "Failed to create task: " + err.Error(),
					}
				}
			} else if clientTask.UpdatedAt.After(serverTask.UpdatedAt) {
				// Client has newer version - update task
				_, err := c.taskRepo.UpdateTask(ctx, clientTask.ID, clientTask.TaskRecord)
				if err != nil {
					return httpError{
						code: http.StatusInternalServerError,
						msg:  "Failed to update task: " + err.Error(),
					}
				}
			}
		}

		return nil
	}
}
