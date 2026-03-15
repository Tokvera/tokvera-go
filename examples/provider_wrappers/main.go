package main

import (
	"context"
	"log"

	tokvera "github.com/tokvera/tokvera-go"
)

func main() {
	tracer := tokvera.NewTracer(tokvera.TrackOptions{
		APIKey:         "tok_live_...",
		Feature:        "multi_model_router",
		TenantID:       "tenant_demo",
		CaptureContent: true,
	})

	ctx := context.Background()
	root, err := tracer.StartTrace(ctx, tokvera.TrackOptions{StepName: "router"})
	if err != nil {
		log.Fatal(err)
	}

	_, err = tracer.TrackOpenAI(ctx, root, tokvera.ProviderRequest{
		Model: "gpt-4o-mini",
		Input: map[string]any{"prompt": "Classify the issue and draft a reply."},
	}, func(context.Context) (tokvera.ProviderResult, error) {
		return tokvera.ProviderResult{
			Output: map[string]any{"text": "OpenAI draft reply"},
			Usage:  tokvera.Usage{PromptTokens: 24, CompletionTokens: 48, TotalTokens: 72},
		}, nil
	})
	if err != nil {
		log.Fatal(err)
	}

	_, err = tracer.TrackMistral(ctx, root, tokvera.ProviderRequest{
		Model: "mistral-small-latest",
		Input: map[string]any{"prompt": "Summarize the reply in one sentence."},
	}, func(context.Context) (tokvera.ProviderResult, error) {
		return tokvera.ProviderResult{
			Output: map[string]any{"text": "One-sentence summary"},
			Usage:  tokvera.Usage{PromptTokens: 18, CompletionTokens: 20, TotalTokens: 38},
		}, nil
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := tracer.FinishSpan(ctx, root, tokvera.FinishSpanOptions{Outcome: "success"}); err != nil {
		log.Fatal(err)
	}
}
