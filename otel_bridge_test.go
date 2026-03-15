package tokvera

import (
	"context"
	"testing"
	"time"
)

func TestOTelBridgeExportsCanonicalEvent(t *testing.T) {
	recorder := newIngestRecorder(t)
	defer recorder.Close()

	bridge := NewOTelBridge(TrackOptions{
		APIKey:   "tok_test_key",
		BaseURL:  recorder.URL(),
		Feature:  "assistant",
		TenantID: "tenant_123",
	})

	err := bridge.Export(context.Background(), []OTelReadableSpan{{
		Name:       "llm_call",
		TraceID:    "trc_otel",
		SpanID:     "spn_otel",
		StartTime:  time.Unix(1700000000, 0),
		EndTime:    time.Unix(1700000000, 250000000),
		StatusCode: "ok",
		Attributes: map[string]any{
			"llm.provider":              "openai",
			"gen_ai.request.model":      "gpt-4o-mini",
			"tokvera.event_type":        "openai.request",
			"tokvera.endpoint":          "responses.create",
			"tokvera.step_name":         "llm_call",
			"gen_ai.usage.total_tokens": int64(17),
		},
	}})
	if err != nil {
		t.Fatalf("export spans: %v", err)
	}

	events := recorder.Events()
	if len(events) != 1 {
		t.Fatalf("expected one otel event, got %d", len(events))
	}
	if events[0].Provider != "openai" {
		t.Fatalf("expected mapped provider, got %s", events[0].Provider)
	}
	if events[0].Metrics == nil || events[0].Metrics.TotalTokens != 17 {
		t.Fatalf("expected mapped token metrics")
	}
	if events[0].Usage.TotalTokens != 17 {
		t.Fatalf("expected mapped usage totals")
	}
	if events[0].Status != "success" {
		t.Fatalf("expected success status, got %s", events[0].Status)
	}
	if events[0].Tags.TraceID != "trc_otel" || events[0].Tags.SpanID != "spn_otel" {
		t.Fatalf("expected otel trace context tags to be preserved")
	}
}
