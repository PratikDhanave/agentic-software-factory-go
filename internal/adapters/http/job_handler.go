package http

import (
	"encoding/json"
	"net/http"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/application/jobs"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

type JobHandler struct {
	getJobStatusUC *jobs.GetJobStatusUseCase
	logger         services.Logger
}

func NewJobHandler(getJobStatusUC *jobs.GetJobStatusUseCase, logger services.Logger) *JobHandler {
	return &JobHandler{
		getJobStatusUC: getJobStatusUC,
		logger:         logger,
	}
}

func (h *JobHandler) HandleGetJobStatus(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		http.Error(w, "missing job id parameter", http.StatusBadRequest)
		return
	}

	job, err := h.getJobStatusUC.Execute(r.Context(), jobID)
	if err != nil {
		h.logger.Error("failed to get job status", err, "id", jobID)
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(job)
}
