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
	"github.com/benjamonnguyen/daygo/sqlite"
	"github.com/google/uuid"
)

type SyncController interface {
	Sync(http.ResponseWriter, *http.Request)
}

type controller struct {
	transactor transactor.Transactor
	taskRepo   daygo.TaskRepo
	logger     daygo.Logger
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
	var syncReq daygo.SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&syncReq); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	c.logger.Info("Sync", "request", syncReq)

	// Process client tasks with conflict resolution within transaction
	toServerSyncCount := 0
	err := c.transactor.WithinTransaction(r.Context(), func(ctx context.Context) error {
		cnt, err := c.syncClientTasks(ctx, syncReq.ClientTasks)
		toServerSyncCount = cnt
		return err
	})
	if c.logAndWriteError(w, err) {
		return
	}

	// Return server tasks to client
	serverTasks, err := c.taskRepo.GetByCreateTime(r.Context(), syncReq.LastSyncTime, time.Time{})
	if err != nil {
		httpErr := httpError{
			msg: "failed to get server tasks: " + err.Error(),
		}
		c.logAndWriteError(w, httpErr)
		return
	}
	response := daygo.SyncResponse{
		ServerTasks:       serverTasks,
		ToServerSyncCount: toServerSyncCount,
	}
	c.logger.Info("Sync", "response", response)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (c *controller) logAndWriteError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	var httpErr httpError
	if errors.As(err, &httpErr) {
		http.Error(w, httpErr.msg, httpErr.code)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	c.logger.Error(err.Error())
	return true
}

func (c *controller) syncClientTasks(ctx context.Context, tasks []daygo.ExistingTaskRecord) (int, error) {
	var existingTaskIDs []any
	for _, clientTask := range tasks {
		if clientTask.ID != uuid.Nil {
			existingTaskIDs = append(existingTaskIDs, clientTask.ID.String())
		}
	}

	var existingTasks []daygo.ExistingTaskRecord
	if len(existingTaskIDs) > 0 {
		existing, err := c.taskRepo.GetTasks(ctx, existingTaskIDs)
		if err != nil && !errors.Is(err, sqlite.ErrNotFound) {
			return 0, httpError{
				code: http.StatusInternalServerError,
				msg:  "Failed getting existing tasks: " + err.Error(),
			}
		}
		existingTasks = existing
	}

	taskIDToExistingRecord := make(map[string]daygo.ExistingTaskRecord)
	for _, task := range existingTasks {
		taskIDToExistingRecord[task.ID.String()] = task
	}

	var cnt int
	for _, clientTask := range tasks {
		serverTask, exists := taskIDToExistingRecord[clientTask.ID.String()]
		if clientTask.ID == uuid.Nil || !exists {
			// New task - create it
			_, err := c.taskRepo.InsertTask(ctx, clientTask.TaskRecord)
			if err != nil {
				return 0, httpError{
					code: http.StatusInternalServerError,
					msg:  "Failed to create task: " + err.Error(),
				}
			}
			cnt += 1
		} else if clientTask.UpdatedAt.After(serverTask.UpdatedAt) {
			// Client has newer version - update task
			_, err := c.taskRepo.UpdateTask(ctx, clientTask.ID, clientTask.TaskRecord)
			if err != nil {
				return 0, httpError{
					code: http.StatusInternalServerError,
					msg:  "Failed to update task: " + err.Error(),
				}
			}
			cnt += 1
		}
	}

	return cnt, nil
}
