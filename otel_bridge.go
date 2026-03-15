package tokvera

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type OTelBridge struct {
	tracer *Tracer
}

func NewOTelBridge(base TrackOptions) *OTelBridge {
	return &OTelBridge{tracer: NewTracer(base)}
}

func (bridge *OTelBridge) Export(ctx context.Context, spans []OTelReadableSpan) error {
	for _, span := range spans {
		handle := TraceHandle{
			TraceID:      chooseString(span.TraceID, generateID("trc")),
			RunID:        attributeString(span.Attributes, "tokvera.run_id"),
			SpanID:       chooseString(span.SpanID, generateID("spn")),
			ParentSpanID: span.ParentSpanID,
			StartedAt:    chooseTime(span.StartTime, time.Now().UTC()),
			Provider: chooseString(
				attributeString(span.Attributes, "tokvera.provider"),
				attributeString(span.Attributes, "llm.provider"),
				"opentelemetry",
			),
			EventType: chooseString(attributeString(span.Attributes, "tokvera.event_type"), "otel_span"),
			Endpoint:  chooseString(attributeString(span.Attributes, "tokvera.endpoint"), "otel"),
			Model: chooseString(
				attributeString(span.Attributes, "tokvera.model"),
				attributeString(span.Attributes, "gen_ai.request.model"),
			),
			Options: TrackOptions{
				Feature:        chooseString(attributeString(span.Attributes, "tokvera.feature"), bridge.tracer.base.Feature),
				TenantID:       chooseString(attributeString(span.Attributes, "tokvera.tenant_id"), bridge.tracer.base.TenantID),
				CustomerID:     chooseString(attributeString(span.Attributes, "tokvera.customer_id"), bridge.tracer.base.CustomerID),
				Environment:    chooseString(attributeString(span.ResourceAttributes, "deployment.environment"), bridge.tracer.base.Environment),
				ConversationID: attributeString(span.Attributes, "tokvera.conversation_id"),
				StepName:       chooseString(attributeString(span.Attributes, "tokvera.step_name"), span.Name),
				SpanKind:       chooseString(attributeString(span.Attributes, "tokvera.span_kind"), "orchestrator"),
				SchemaVersion:  chooseString(bridge.tracer.base.SchemaVersion, TraceSchemaVersionV2),
			},
		}
		if handle.RunID == "" {
			handle.RunID = handle.TraceID
		}
		handle.Options.TraceID = handle.TraceID
		handle.Options.RunID = handle.RunID
		handle.Options.SpanID = handle.SpanID
		handle.Options.ParentSpanID = handle.ParentSpanID
		handle.Options.Provider = handle.Provider
		handle.Options.EventType = handle.EventType
		handle.Options.Endpoint = handle.Endpoint
		handle.Options.Model = handle.Model

		metrics := &TraceMetrics{
			PromptTokens:     attributeInt(span.Attributes, "gen_ai.usage.prompt_tokens"),
			CompletionTokens: attributeInt(span.Attributes, "gen_ai.usage.completion_tokens"),
			TotalTokens:      attributeInt(span.Attributes, "gen_ai.usage.total_tokens"),
			LatencyMs:        span.EndTime.Sub(handle.StartedAt).Milliseconds(),
		}
		options := FinishSpanOptions{Metrics: metrics}

		if strings.EqualFold(span.StatusCode, "error") {
			options.Error = &EventError{
				Message: chooseString(span.StatusDescription, "otel span failed"),
				Type:    "otel_error",
			}
			if err := bridge.tracer.FailSpan(ctx, handle, fmt.Errorf(options.Error.Message), options); err != nil {
				return err
			}
			continue
		}
		if err := bridge.tracer.FinishSpan(ctx, handle, options); err != nil {
			return err
		}
	}
	return nil
}

func attributeString(values map[string]any, key string) string {
	if len(values) == 0 {
		return ""
	}
	value, ok := values[key]
	if !ok {
		return ""
	}
	stringValue, ok := value.(string)
	if !ok {
		return ""
	}
	return stringValue
}

func attributeInt(values map[string]any, key string) int {
	if len(values) == 0 {
		return 0
	}
	value, ok := values[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func chooseTime(value time.Time, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value
}
