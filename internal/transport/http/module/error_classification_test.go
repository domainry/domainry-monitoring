package module

import (
	"net/http/httptest"
	"testing"
)

type classificationCapture struct {
	*httptest.ResponseRecorder
	kind, code string
}

func (capture *classificationCapture) RecordError(kind, code string) {
	capture.kind, capture.code = kind, code
}

func TestMonitoringErrorClassificationMatrix(t *testing.T) {
	tests := []struct {
		status int
		kind   string
		code   string
	}{
		{503, "unavailable", "backend.monitoring.database_unavailable"},
		{502, "unavailable", "backend.monitoring.collector_unavailable"},
		{409, "conflict", "backend.monitoring.aggregation_fence_rejected"},
		{400, "bad_request", "backend.monitoring.query_invalid"},
		{403, "forbidden", "backend.monitoring.metrics_scope_denied"},
	}
	for _, test := range tests {
		capture := &classificationCapture{ResponseRecorder: httptest.NewRecorder()}
		writeModuleError(capture, test.status, test.code)
		if capture.kind != test.kind || capture.code != test.code {
			t.Fatalf("status=%d kind=%q code=%q", test.status, capture.kind, capture.code)
		}
	}
}
