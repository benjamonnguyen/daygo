package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/benjamonnguyen/daygo"
)

type controller struct {
	taskRepo daygo.TaskRepo
}

func (c *controller) Sync(w http.ResponseWriter, r *http.Request) {
	// Get all tasks from database
	tasks, err := c.taskRepo.GetByStartTime(r.Context(), time.Time{}, time.Time{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get tasks: %v", err), http.StatusInternalServerError)
		return
	}

	// Return tasks as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

func handleSyncPost(w http.ResponseWriter, r *http.Request, taskRepo daygo.TaskRepo) {
	// Parse incoming JSON data
	var syncData struct {
		Tasks []daygo.ExistingTaskRecord `json:"tasks"`
	}

	if err := json.NewDecoder(r.Body).Decode(&syncData); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Process each task (this is a simple implementation - you might want more sophisticated sync logic)
	var results []string
	for _, task := range syncData.Tasks {
		// Here you would implement your sync logic
		// For now, just acknowledge receipt
		results = append(results, fmt.Sprintf("Received task: %s", task.Name))
	}

	// Return success response
	response := map[string]interface{}{
		"status":  "success",
		"message": "Sync completed",
		"results": results,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}
