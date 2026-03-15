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
	if events[0].SpanKind != "model" {
		t.Fatalf("expected model span kind, got %s", events[0].SpanKind)
	}
	if len(events[0].PayloadBlocks) < 2 {
		t.Fatalf("expected input and output payload blocks, got %d", len(events[0].PayloadBlocks))
	}
}
