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
	if events[0].Provider != "tokvera" {
		t.Fatalf("expected tokvera provider, got %s", events[0].Provider)
	}
	if events[0].EventType != "tokvera.trace" {
		t.Fatalf("expected tokvera.trace event type, got %s", events[0].EventType)
	}
	if events[0].Endpoint != "manual.trace" {
		t.Fatalf("expected manual.trace endpoint, got %s", events[0].Endpoint)
	}
	if events[1].Status != "success" {
		t.Fatalf("expected success, got %s", events[1].Status)
	}
	if events[1].Tags.TraceID == "" || events[1].Tags.RunID == "" || events[1].Tags.SpanID == "" {
		t.Fatalf("expected trace context to be populated")
	}
	if len(events[1].PayloadBlocks) != 1 {
		t.Fatalf("expected one payload block, got %d", len(events[1].PayloadBlocks))
	}
	if events[1].Usage.TotalTokens != 20 {
		t.Fatalf("expected usage to round-trip, got %d", events[1].Usage.TotalTokens)
	}
	if events[1].Tags.Outcome != "success" {
		t.Fatalf("expected success outcome tag, got %s", events[1].Tags.Outcome)
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
	if options.SpanID != "" {
		t.Fatalf("expected child span id to be generated later, got %s", options.SpanID)
	}
	if options.Provider != "openai" {
		t.Fatalf("expected override provider to win, got %s", options.Provider)
	}
	if options.Model != "gpt-4o-mini" {
		t.Fatalf("expected model inheritance, got %s", options.Model)
	}
}

func TestStartSpanGeneratesDistinctSpanID(t *testing.T) {
	recorder := newIngestRecorder(t)
	defer recorder.Close()

	tracer := NewTracer(TrackOptions{
		APIKey:              "tok_test_key",
		BaseURL:             recorder.URL(),
		Feature:             "assistant",
		TenantID:            "tenant_123",
		EmitLifecycleEvents: true,
	})

	root, err := tracer.StartTrace(context.Background(), TrackOptions{StepName: "root_flow"})
	if err != nil {
		t.Fatalf("start trace: %v", err)
	}

	child, err := tracer.StartSpan(context.Background(), root, TrackOptions{
		Provider:  "openai",
		EventType: "openai.request",
		Endpoint:  "responses.create",
		StepName:  "draft_reply",
		SpanKind:  "model",
	})
	if err != nil {
		t.Fatalf("start span: %v", err)
	}

	if child.SpanID == root.SpanID {
		t.Fatalf("expected child span id to differ from root span id")
	}
	if child.ParentSpanID != root.SpanID {
		t.Fatalf("expected parent span id %s, got %s", root.SpanID, child.ParentSpanID)
	}
	if child.TraceID != root.TraceID || child.RunID != root.RunID {
		t.Fatalf("expected child trace context to stay within root trace")
	}
}
