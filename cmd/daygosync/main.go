package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/charmbracelet/log"

	"github.com/benjamonnguyen/daygo"
	"github.com/benjamonnguyen/daygo/sqlite"
	dsdb "github.com/benjamonnguyen/deadsimple/database/sqlite"
)

func main() {
	confDir, _ := os.UserConfigDir()
	cfg, err := LoadConf(path.Join(confDir, "daygo", "daygosync.conf"))
	if err != nil {
		panic(err)
	}

	// logger
	var logPath, logLvl string
	if err := cfg.Get(KeyLogPath, &logPath); err != nil {
		panic(err)
	}
	if err := cfg.Get(KeyLogLevel, &logLvl); err != nil {
		panic(err)
	}

	var logger daygo.Logger
	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE, 0o644)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		logger = log.New(f, "", 0)
	} else {
		logger = log.New(os.Stdout, "", 0)
	}

	// db
	var dbUrl string
	if err := cfg.Get(KeyDbUrl, &dbUrl); err != nil {
		panic(err)
	}

	db, err := dsdb.Open(dbUrl)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// repos
	taskRepo := sqlite.NewTaskRepo(db.Conn(), logger)

	// routes
	c := &controller{
		taskRepo: taskRepo,
	}
	http.HandleFunc("GET /sync", c.Sync)

	// Start the server
	var port string
	if err := cfg.Get(KeyPort, &port); err != nil {
		panic(err)
	}

	logger.Info("Starting sync server on port %s", port)
	logger.Fatal(http.ListenAndServe(":"+port, nil))
}

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

