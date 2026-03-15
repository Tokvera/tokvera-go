package main

import (
	"context"
	"log"

	tokvera "github.com/tokvera/tokvera-go"
	"github.com/tokvera/tokvera-go/examples/internal/exampleenv"
)

func main() {
	tracer := tokvera.NewTracer(exampleenv.BaseOptions("go_manual_tracer", true))

	ctx := context.Background()
	root, err := tracer.StartTrace(ctx, tokvera.TrackOptions{
		StepName: "request_flow",
		SpanKind: "orchestrator",
	})
	if err != nil {
		log.Fatal(err)
	}

	modelSpan, err := tracer.StartSpan(ctx, root, tokvera.TrackOptions{
		Provider:  "openai",
		EventType: "openai.request",
		Endpoint:  "responses.create",
		Model:     "gpt-4o-mini",
		StepName:  "draft_reply",
		SpanKind:  "model",
	})
	if err != nil {
		log.Fatal(err)
	}
	modelSpan, err = tracer.AttachPayload(modelSpan, map[string]any{
		"messages": []map[string]string{{"role": "user", "content": "Explain retry loops in traces."}},
	}, "prompt_input")
	if err != nil {
		log.Fatal(err)
	}
	if err := tracer.FinishSpan(ctx, modelSpan, tokvera.FinishSpanOptions{
		Usage:   tokvera.Usage{PromptTokens: 54, CompletionTokens: 121, TotalTokens: 175},
		Metrics: &tokvera.TraceMetrics{CostUSD: 0.00042},
		Outcome: "success",
	}); err != nil {
		log.Fatal(err)
	}
	if err := tracer.FinishSpan(ctx, root, tokvera.FinishSpanOptions{Outcome: "success"}); err != nil {
		log.Fatal(err)
	}
}
