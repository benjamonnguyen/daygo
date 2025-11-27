package main

import (
	"encoding/json"
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
	LastSyncedAt time.Time
	ClientTasks  []daygo.ExistingTaskRecord
}

type SyncResponse struct {
	ServerTasks []daygo.ExistingTaskRecord
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

	// Get tasks from server that were updated since the provided timestamp
	serverTasks, err := c.taskRepo.GetByStartTime(r.Context(), syncReq.LastSyncedAt, time.Now())
	if err != nil {
		http.Error(w, "Failed to get server tasks: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Process client tasks with conflict resolution
	var existingTaskIDs []any
	for _, clientTask := range syncReq.ClientTasks {
		if clientTask.ID > 0 {
			existingTaskIDs = append(existingTaskIDs, clientTask.ID)
		}
	}

	// Get existing tasks from server to check for conflicts
	existingTasks, err := c.taskRepo.GetTasks(r.Context(), existingTaskIDs)
	if err != nil {
		http.Error(w, "Failed to get existing tasks"+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a map for quick lookup of existing tasks by ID
	existingTasksMap := make(map[int]daygo.ExistingTaskRecord)
	for _, task := range existingTasks {
		existingTasksMap[task.ID] = task
	}

	// Process each client task with conflict resolution
	for _, clientTask := range syncReq.ClientTasks {
		if clientTask.ID == 0 {
			// New task - create it
			_, err := c.taskRepo.InsertTask(r.Context(), clientTask.TaskRecord)
			if err != nil {
				http.Error(w, "Failed to create task: "+err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			// Existing task - check for conflicts
			if serverTask, exists := existingTasksMap[clientTask.ID]; exists {
				// Task exists on server - check if server has newer version
				if clientTask.UpdatedAt.Before(serverTask.UpdatedAt) {
					// Server has newer version - skip client update
					continue
				}
				// Client has newer or same version - update task
				_, err := c.taskRepo.UpdateTask(r.Context(), clientTask.ID, clientTask.TaskRecord)
				if err != nil {
					http.Error(w, "Failed to update task: "+err.Error(), http.StatusInternalServerError)
					return
				}
			} else {
				// Task no longer exists on server - create it
				_, err := c.taskRepo.InsertTask(r.Context(), clientTask.TaskRecord)
				if err != nil {
					http.Error(w, "Failed to recreate task: "+err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
	}

	// Return server tasks to client
	response := SyncResponse{
		ServerTasks: serverTasks,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
	}
}
