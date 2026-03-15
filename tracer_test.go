package tokvera

import (
	"context"
	"testing"
)

func TestTracerStartFinishLifecycle(t *testing.T) {
	recorder := newIngestRecorder(t)
	defer recorder.Close()

	tracer := NewTracer(TrackOptions{
		APIKey:              "tok_test_key",
		BaseURL:             recorder.URL(),
		Feature:             "assistant",
		TenantID:            "tenant_123",
		CaptureContent:      true,
		EmitLifecycleEvents: true,
	})

	handle, err := tracer.StartTrace(context.Background(), TrackOptions{StepName: "root_flow"})
	if err != nil {
		t.Fatalf("start trace: %v", err)
	}
	handle, err = tracer.AttachPayload(handle, map[string]any{"prompt": "hello"}, "prompt_input")
	if err != nil {
		t.Fatalf("attach payload: %v", err)
	}
	if err := tracer.FinishSpan(context.Background(), handle, FinishSpanOptions{
		Usage:   Usage{PromptTokens: 12, CompletionTokens: 8, TotalTokens: 20},
		Outcome: "success",
	}); err != nil {
		t.Fatalf("finish span: %v", err)
	}

	events := recorder.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Status != "in_progress" {
		t.Fatalf("expected in_progress, got %s", events[0].Status)
	}
	if events[1].Status != "success" {
		t.Fatalf("expected success, got %s", events[1].Status)
	}
	if events[1].TraceID == "" || events[1].RunID == "" || events[1].SpanID == "" {
		t.Fatalf("expected trace context to be populated")
	}
	if len(events[1].PayloadBlocks) != 1 {
		t.Fatalf("expected one payload block, got %d", len(events[1].PayloadBlocks))
	}
	if events[1].Usage.TotalTokens != 20 {
		t.Fatalf("expected usage to round-trip, got %d", events[1].Usage.TotalTokens)
	}
}

func TestTrackOptionsFromTraceContextUsesParentIDs(t *testing.T) {
	tracer := NewTracer(TrackOptions{APIKey: "tok_test_key", Feature: "assistant", TenantID: "tenant_123"})
	handle := TraceHandle{
		TraceID: "trc_existing",
		RunID:   "run_existing",
		SpanID:  "spn_parent",
		Model:   "gpt-4o-mini",
		Options: TrackOptions{Feature: "assistant"},
	}

	options := tracer.TrackOptionsFromTraceContext(handle, TrackOptions{Provider: "openai"})
	if options.TraceID != "trc_existing" {
		t.Fatalf("expected trace id inheritance, got %s", options.TraceID)
	}
	if options.RunID != "run_existing" {
		t.Fatalf("expected run id inheritance, got %s", options.RunID)
	}
	if options.ParentSpanID != "spn_parent" {
		t.Fatalf("expected parent span id inheritance, got %s", options.ParentSpanID)
	}
	if options.Provider != "openai" {
		t.Fatalf("expected override provider to win, got %s", options.Provider)
	}
	if options.Model != "gpt-4o-mini" {
		t.Fatalf("expected model inheritance, got %s", options.Model)
	}
}
