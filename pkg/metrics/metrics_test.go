package metrics

import (
	"testing"
	"time"
)

func TestMetricsRecording(t *testing.T) {
	mc := NewMetricsCollector(100)

	mc.RecordWorkflowStart("wf-1", "spec-to-code")
	
	if mc.metrics.WorkflowsStarted != 1 {
		t.Error("Workflow start not recorded")
	}

	mc.RecordWorkflowCompletion("wf-1", true, 1*time.Second, 2, 85, 90)
	
	if mc.metrics.WorkflowsCompleted != 1 {
		t.Error("Workflow completion not recorded")
	}

	if mc.metrics.BugsDetected != 2 {
		t.Error("Bugs not recorded")
	}
}

func TestRequestMetrics(t *testing.T) {
	mc := NewMetricsCollector(100)

	mc.RecordRequest("GET", "/api/health", 200, 10*time.Millisecond, "")
	mc.RecordRequest("POST", "/api/workflows", 201, 50*time.Millisecond, "")
	mc.RecordRequest("GET", "/api/error", 500, 100*time.Millisecond, "Internal error")

	if mc.metrics.RequestsTotal != 3 {
		t.Error("Request count incorrect")
	}

	if mc.metrics.RequestsError != 1 {
		t.Error("Error count incorrect")
	}
}

func TestMetricsGetMetrics(t *testing.T) {
	mc := NewMetricsCollector(100)
	mc.RecordWorkflowStart("wf-1", "test")
	mc.RecordWorkflowCompletion("wf-1", true, 1*time.Second, 1, 80, 85)

	metrics := mc.GetMetrics()

	wf := metrics["workflows"].(map[string]interface{})
	if wf["completed"] != int64(1) {
		t.Error("Completed count incorrect in metrics")
	}
}
