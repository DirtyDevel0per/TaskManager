package handler

import (
	"encoding/json"
	"net/http"
	"task-manager/internal/handler/middleware"
	"task-manager/internal/models"
)

func (h *TaskHandler) BatchCreateTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Tasks []models.CreateTaskRequest `json:"tasks"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Tasks) == 0 {
		http.Error(w, "At least one task is required", http.StatusBadRequest)
		return
	}

	if len(req.Tasks) > 100 {
		http.Error(w, "Maximum 100 tasks per batch", http.StatusBadRequest)
		return
	}

	count, err := h.taskService.BatchCreate(userID, req.Tasks)
	if err != nil {
		http.Error(w, "Failed to queue tasks", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Tasks queued for creation",
		"count":   count,
	})
}

func (h *TaskHandler) ExportTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	if format != "csv" && format != "json" {
		http.Error(w, "Unsupported format. Use csv or json", http.StatusBadRequest)
		return
	}

	jobID, err := h.taskService.ExportTasks(userID, format)
	if err != nil {
		http.Error(w, "Failed to start export", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"job_id":  jobID,
		"status":  "processing",
		"message": "Export started, you will be notified when complete",
	})
}
