# tokvera-go

Go SDK scaffold for Tokvera's AI cost and trace intelligence platform.

## Status

This repository is a **CI-first Wave 1 scaffold**.

It already includes the intended public shape for:
- manual trace and span lifecycle
- provider-operation wrappers for OpenAI, Anthropic, Gemini, and Mistral
- payload attachment
- live lifecycle events for `/dashboard/traces/live`
- a generic OpenTelemetry bridge surface
- runnable examples and unit tests

It is **not official yet**. Promotion is blocked until all of these pass:
- Go CI (`gofmt`, `go vet`, `go test`)
- canonical contract verification against the deployed API
- lifecycle visibility in `/dashboard/traces/live`
- trace detail and inspector visibility in the dashboard
- mixed-composition duplicate-emission checks

## Install

```bash
go get github.com/tokvera/tokvera-go
```

## Manual Tracer

```go
package main

import (
  "context"
  "log"

  tokvera "github.com/tokvera/tokvera-go"
)

func main() {
  tracer := tokvera.NewTracer(tokvera.TrackOptions{
    APIKey:              "tok_live_...",
    Feature:             "support_assistant",
    TenantID:            "tenant_123",
    CaptureContent:      true,
    EmitLifecycleEvents: true,
  })

  ctx := context.Background()
  root, err := tracer.StartTrace(ctx, tokvera.TrackOptions{StepName: "request_flow"})
  if err != nil {
    log.Fatal(err)
  }

  child, err := tracer.StartSpan(ctx, root, tokvera.TrackOptions{
    Provider:  "openai",
    EventType: "responses_create",
    Endpoint:  "/v1/responses",
    Model:     "gpt-4o-mini",
    StepName:  "draft_reply",
    SpanKind:  "model",
  })
  if err != nil {
    log.Fatal(err)
  }

  child, err = tracer.AttachPayload(child, map[string]any{"prompt": "Explain retry loops."}, "prompt_input")
  if err != nil {
    log.Fatal(err)
  }

  if err := tracer.FinishSpan(ctx, child, tokvera.FinishSpanOptions{
    Usage: tokvera.Usage{PromptTokens: 24, CompletionTokens: 48, TotalTokens: 72},
    Outcome: "success",
  }); err != nil {
    log.Fatal(err)
  }

  if err := tracer.FinishSpan(ctx, root, tokvera.FinishSpanOptions{Outcome: "success"}); err != nil {
    log.Fatal(err)
  }
}
```

## Provider Wrappers

Use the provider helpers when you already have a root trace or orchestrator span and want child model spans without losing one coherent trace.

```go
_, err := tracer.TrackOpenAI(ctx, root, tokvera.ProviderRequest{
  Model: "gpt-4o-mini",
  Input: map[string]any{"prompt": "Classify this support issue."},
}, func(context.Context) (tokvera.ProviderResult, error) {
  return tokvera.ProviderResult{
    Output: map[string]any{"text": "billing"},
    Usage:  tokvera.Usage{PromptTokens: 10, CompletionTokens: 2, TotalTokens: 12},
  }, nil
})
```

Matching helpers exist for:
- `TrackAnthropic`
- `TrackGemini`
- `TrackMistral`

## OpenTelemetry Bridge

The bridge converts generic readable spans into Tokvera's canonical trace events.

```go
bridge := tokvera.NewOTelBridge(tokvera.TrackOptions{
  APIKey:   "tok_live_...",
  Feature:  "assistant",
  TenantID: "tenant_123",
})

err := bridge.Export(ctx, []tokvera.OTelReadableSpan{
  {
    Name:       "llm_call",
    TraceID:    "trc_otel",
    SpanID:     "spn_otel",
    StartTime:  time.Now().Add(-250 * time.Millisecond),
    EndTime:    time.Now(),
    StatusCode: "ok",
    Attributes: map[string]any{
      "llm.provider":          "openai",
      "gen_ai.request.model":  "gpt-4o-mini",
      "tokvera.event_type":    "responses_create",
      "gen_ai.usage.total_tokens": int64(17),
    },
  },
})
```

## Examples

- `examples/manual_tracer`
- `examples/provider_wrappers`

## Local Development

```bash
gofmt -w .
go vet ./...
go test ./...
node scripts/check-canonical-contract.mjs
```

## Release Bar

Before `tokvera-go` can be marked complete in the execution board:
- docs page exists and is linked from Tokvera docs
- examples run end-to-end
- canonical contract checks pass
- live traces show lifecycle rows for Go events
- dashboard visibility passes in overview, traces, live traces, and trace detail
