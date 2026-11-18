package http

import (
	"encoding/json"
	"net/http"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/application/jobs"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/infrastructure/observability"
)

type JobHandler struct {
	getJobStatusUC *jobs.GetJobStatusUseCase
	logger         services.Logger
	observer       *observability.UseCaseObserver
}

func NewJobHandler(getJobStatusUC *jobs.GetJobStatusUseCase, logger services.Logger, observer *observability.UseCaseObserver) *JobHandler {
	return &JobHandler{
		getJobStatusUC: getJobStatusUC,
		logger:         logger,
		observer:       observer,
	}
}

func (h *JobHandler) HandleGetJobStatus(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		http.Error(w, "missing job id parameter", http.StatusBadRequest)
		return
	}

	var job interface{}
	err := h.observer.ObserveExecution(r.Context(), "get-job-status", func(span observability.SpanContext) error {
		span.SetAttribute("job_id", jobID)
		var execErr error
		job, execErr = h.getJobStatusUC.Execute(r.Context(), jobID)
		return execErr
	})

	if err != nil {
		h.logger.Error("failed to get job status", err, "id", jobID)
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(job)
}
