package tokvera

import (
	"context"
	"testing"
)

func TestTrackOpenAICreatesModelSpan(t *testing.T) {
	recorder := newIngestRecorder(t)
	defer recorder.Close()

	tracer := NewTracer(TrackOptions{
		APIKey:         "tok_test_key",
		BaseURL:        recorder.URL(),
		Feature:        "assistant",
		TenantID:       "tenant_123",
		CaptureContent: true,
	})
	root, err := tracer.StartTrace(context.Background(), TrackOptions{StepName: "workflow"})
	if err != nil {
		t.Fatalf("start trace: %v", err)
	}

	_, err = tracer.TrackOpenAI(context.Background(), root, ProviderRequest{
		Model: "gpt-4o-mini",
		Input: map[string]any{"messages": []string{"hello"}},
	}, func(context.Context) (ProviderResult, error) {
		return ProviderResult{
			Output: map[string]any{"text": "world"},
			Usage:  Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5},
		}, nil
	})
	if err != nil {
		t.Fatalf("track openai: %v", err)
	}

	events := recorder.Events()
	if len(events) != 1 {
		t.Fatalf("expected one provider event, got %d", len(events))
	}
	if events[0].Provider != "openai" {
		t.Fatalf("expected openai provider, got %s", events[0].Provider)
	}
	if events[0].Tags.ParentSpanID != root.SpanID {
		t.Fatalf("expected provider span parent %s, got %s", root.SpanID, events[0].Tags.ParentSpanID)
	}
	if events[0].Tags.SpanID == root.SpanID {
		t.Fatalf("expected provider span id to differ from root span id")
	}
	if events[0].SpanKind != "model" {
		t.Fatalf("expected model span kind, got %s", events[0].SpanKind)
	}
	if len(events[0].PayloadBlocks) < 2 {
		t.Fatalf("expected input and output payload blocks, got %d", len(events[0].PayloadBlocks))
	}
}

func TestManualTracerComposesWithOpenAIWithoutDuplicateSpanStatuses(t *testing.T) {
	recorder := newIngestRecorder(t)
	defer recorder.Close()

	tracer := NewTracer(TrackOptions{
		APIKey:              "tok_test_key",
		BaseURL:             recorder.URL(),
		Feature:             "mixed_manual_openai",
		TenantID:            "tenant_123",
		CaptureContent:      true,
		EmitLifecycleEvents: true,
	})

	root, err := tracer.StartTrace(context.Background(), TrackOptions{
		StepName: "gateway_request",
		SpanKind: "orchestrator",
		Model:    "router",
	})
	if err != nil {
		t.Fatalf("start trace: %v", err)
	}

	_, err = tracer.TrackOpenAI(context.Background(), root, ProviderRequest{
		Model: "gpt-4o-mini",
		Input: map[string]any{"messages": []string{"hello"}},
	}, func(context.Context) (ProviderResult, error) {
		return ProviderResult{
			Output: map[string]any{"text": "world"},
			Usage:  Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5},
		}, nil
	})
	if err != nil {
		t.Fatalf("track openai: %v", err)
	}

	if err := tracer.FinishSpan(context.Background(), root, FinishSpanOptions{Outcome: "success"}); err != nil {
		t.Fatalf("finish root: %v", err)
	}

	events := recorder.Events()
	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(events))
	}

	keys := map[string]struct{}{}
	spanIDs := map[string]struct{}{}
	for _, event := range events {
		key := event.Tags.TraceID + ":" + event.Tags.SpanID + ":" + event.Status
		if _, exists := keys[key]; exists {
			t.Fatalf("duplicate trace/span/status detected: %s", key)
		}
		keys[key] = struct{}{}
		spanIDs[event.Tags.SpanID] = struct{}{}
	}
	if len(spanIDs) != 2 {
		t.Fatalf("expected 2 distinct spans, got %d", len(spanIDs))
	}

	var providerTerminal Event
	for _, event := range events {
		if event.Provider == "openai" && event.Status == "success" {
			providerTerminal = event
			break
		}
	}
	if providerTerminal.Tags.ParentSpanID != root.SpanID {
		t.Fatalf("expected provider span parent %s, got %s", root.SpanID, providerTerminal.Tags.ParentSpanID)
	}
	if providerTerminal.Tags.TraceID != root.TraceID {
		t.Fatalf("expected provider trace %s, got %s", root.TraceID, providerTerminal.Tags.TraceID)
	}
}
