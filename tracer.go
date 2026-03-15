package tokvera

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Tracer struct {
	base   TrackOptions
	client *Client
}

func NewTracer(base TrackOptions) *Tracer {
	clientOptions := []ClientOption{}
	if strings.TrimSpace(base.BaseURL) != "" {
		clientOptions = append(clientOptions, WithBaseURL(base.BaseURL))
	}
	return &Tracer{
		base:   base,
		client: NewClient(base.APIKey, clientOptions...),
	}
}

func (tracer *Tracer) StartTrace(ctx context.Context, options TrackOptions) (TraceHandle, error) {
	merged := mergeTrackOptions(tracer.base, options)
	handle := TraceHandle{
		TraceID:      chooseString(options.TraceID, merged.TraceID, generateID("trc")),
		RunID:        chooseString(options.RunID, merged.RunID, generateID("run")),
		SpanID:       chooseString(options.SpanID, merged.SpanID, generateID("spn")),
		ParentSpanID: "",
		StartedAt:    time.Now().UTC(),
		Provider:     chooseString(options.Provider, merged.Provider, "custom"),
		EventType:    chooseString(options.EventType, merged.EventType, "manual_trace"),
		Endpoint:     chooseString(options.Endpoint, merged.Endpoint, "manual"),
		Model:        chooseString(options.Model, merged.Model),
		Options:      merged,
	}
	handle.Options.TraceID = handle.TraceID
	handle.Options.RunID = handle.RunID
	handle.Options.SpanID = handle.SpanID
	handle.Options.ParentSpanID = ""
	handle.Options.Provider = handle.Provider
	handle.Options.EventType = handle.EventType
	handle.Options.Endpoint = handle.Endpoint
	handle.Options.Model = handle.Model
	handle.Options.StepName = chooseString(options.StepName, merged.StepName, "trace_root")
	handle.Options.SpanKind = chooseString(options.SpanKind, merged.SpanKind, "orchestrator")
	handle.Options.SchemaVersion = chooseString(options.SchemaVersion, merged.SchemaVersion, TraceSchemaVersionV2)

	if handle.Options.EmitLifecycleEvents {
		if err := tracer.client.IngestEvent(ctx, buildEvent(handle, "in_progress", FinishSpanOptions{})); err != nil {
			return TraceHandle{}, err
		}
	}
	return handle, nil
}

func (tracer *Tracer) StartSpan(ctx context.Context, parent TraceHandle, options TrackOptions) (TraceHandle, error) {
	merged := tracer.TrackOptionsFromTraceContext(parent, options)
	handle := TraceHandle{
		TraceID:      chooseString(options.TraceID, merged.TraceID, parent.TraceID),
		RunID:        chooseString(options.RunID, merged.RunID, parent.RunID),
		SpanID:       chooseString(options.SpanID, merged.SpanID, generateID("spn")),
		ParentSpanID: chooseString(options.ParentSpanID, merged.ParentSpanID, parent.SpanID),
		StartedAt:    time.Now().UTC(),
		Provider:     chooseString(options.Provider, merged.Provider, parent.Provider, "custom"),
		EventType:    chooseString(options.EventType, merged.EventType, "manual_span"),
		Endpoint:     chooseString(options.Endpoint, merged.Endpoint, "manual"),
		Model:        chooseString(options.Model, merged.Model, parent.Model),
		Options:      merged,
	}
	handle.Options.TraceID = handle.TraceID
	handle.Options.RunID = handle.RunID
	handle.Options.SpanID = handle.SpanID
	handle.Options.ParentSpanID = handle.ParentSpanID
	handle.Options.Provider = handle.Provider
	handle.Options.EventType = handle.EventType
	handle.Options.Endpoint = handle.Endpoint
	handle.Options.Model = handle.Model
	handle.Options.StepName = chooseString(options.StepName, merged.StepName, "span_step")
	handle.Options.SpanKind = chooseString(options.SpanKind, merged.SpanKind, "orchestrator")
	handle.Options.SchemaVersion = chooseString(options.SchemaVersion, merged.SchemaVersion, TraceSchemaVersionV2)

	if handle.Options.EmitLifecycleEvents {
		if err := tracer.client.IngestEvent(ctx, buildEvent(handle, "in_progress", FinishSpanOptions{})); err != nil {
			return TraceHandle{}, err
		}
	}
	return handle, nil
}

func (tracer *Tracer) FinishSpan(ctx context.Context, handle TraceHandle, options FinishSpanOptions) error {
	return tracer.client.IngestEvent(ctx, buildEvent(handle, "success", options))
}

func (tracer *Tracer) FailSpan(ctx context.Context, handle TraceHandle, cause error, options FinishSpanOptions) error {
	if options.Error == nil {
		options.Error = &EventError{
			Message: chooseString(errorMessage(cause), "span failed"),
			Type:    "runtime_error",
		}
	}
	return tracer.client.IngestEvent(ctx, buildEvent(handle, "failure", options))
}

func (tracer *Tracer) AttachPayload(handle TraceHandle, payload any, payloadType string) (TraceHandle, error) {
	content, err := stringifyPayload(payload)
	if err != nil {
		return TraceHandle{}, err
	}
	updated := handle
	updated.Options.PayloadBlocks = append(updated.Options.PayloadBlocks, TracePayloadBlock{
		PayloadType: chooseString(payloadType, "other"),
		Content:     content,
	})
	return updated, nil
}

func (tracer *Tracer) TrackOptionsFromTraceContext(handle TraceHandle, overrides TrackOptions) TrackOptions {
	merged := mergeTrackOptions(tracer.base, handle.Options)
	merged = mergeTrackOptions(merged, overrides)
	if strings.TrimSpace(overrides.TraceID) == "" {
		merged.TraceID = handle.TraceID
	}
	if strings.TrimSpace(overrides.RunID) == "" {
		merged.RunID = handle.RunID
	}
	if strings.TrimSpace(overrides.ParentSpanID) == "" && strings.TrimSpace(overrides.SpanID) == "" {
		merged.ParentSpanID = handle.SpanID
	}
	if strings.TrimSpace(overrides.Provider) == "" {
		merged.Provider = handle.Provider
	}
	if strings.TrimSpace(overrides.EventType) == "" {
		merged.EventType = handle.EventType
	}
	if strings.TrimSpace(overrides.Endpoint) == "" {
		merged.Endpoint = handle.Endpoint
	}
	if strings.TrimSpace(overrides.Model) == "" {
		merged.Model = handle.Model
	}
	return merged
}

func buildEvent(handle TraceHandle, status string, options FinishSpanOptions) Event {
	metrics := mergeMetrics(handle.Options.Metrics, options.Metrics)
	if metrics == nil {
		metrics = &TraceMetrics{}
	}
	if metrics.LatencyMs == 0 {
		metrics.LatencyMs = time.Since(handle.StartedAt).Milliseconds()
	}
	if metrics.PromptTokens == 0 {
		metrics.PromptTokens = options.Usage.PromptTokens
	}
	if metrics.CompletionTokens == 0 {
		metrics.CompletionTokens = options.Usage.CompletionTokens
	}
	if metrics.TotalTokens == 0 {
		metrics.TotalTokens = options.Usage.TotalTokens
	}

	payloadBlocks := append([]TracePayloadBlock{}, handle.Options.PayloadBlocks...)
	payloadBlocks = append(payloadBlocks, options.PayloadBlocks...)
	decision := mergeDecision(handle.Options.Decision, options.Decision)

	return Event{
		Provider:       handle.Provider,
		EventType:      handle.EventType,
		Endpoint:       handle.Endpoint,
		Model:          handle.Model,
		Status:         status,
		Timestamp:      time.Now().UTC(),
		Feature:        handle.Options.Feature,
		TenantID:       handle.Options.TenantID,
		CustomerID:     handle.Options.CustomerID,
		AttemptType:    handle.Options.AttemptType,
		Plan:           handle.Options.Plan,
		Environment:    handle.Options.Environment,
		TemplateID:     handle.Options.TemplateID,
		TraceID:        handle.TraceID,
		RunID:          handle.RunID,
		ConversationID: handle.Options.ConversationID,
		SpanID:         handle.SpanID,
		ParentSpanID:   handle.ParentSpanID,
		StepName:       handle.Options.StepName,
		Outcome:        chooseString(options.Outcome, handle.Options.Outcome, mapStatusToOutcome(status)),
		RetryReason:    chooseString(handle.Options.RetryReason, decisionRetry(decision)),
		FallbackReason: chooseString(handle.Options.FallbackReason, decisionFallback(decision)),
		QualityLabel:   chooseString(options.QualityLabel, handle.Options.QualityLabel),
		FeedbackScore:  chooseFloatPointer(options.FeedbackScore, handle.Options.FeedbackScore),
		SchemaVersion:  chooseString(handle.Options.SchemaVersion, TraceSchemaVersionV2),
		SpanKind:       handle.Options.SpanKind,
		ToolName:       handle.Options.ToolName,
		PayloadRefs:    append([]string{}, handle.Options.PayloadRefs...),
		PayloadBlocks:  payloadBlocks,
		Metrics:        metrics,
		Decision:       decision,
		Usage:          options.Usage,
		LatencyMs:      metrics.LatencyMs,
		Error:          options.Error,
	}
}

func mergeTrackOptions(base TrackOptions, override TrackOptions) TrackOptions {
	merged := base
	merged.APIKey = chooseString(override.APIKey, base.APIKey)
	merged.BaseURL = chooseString(override.BaseURL, base.BaseURL)
	merged.Feature = chooseString(override.Feature, base.Feature)
	merged.TenantID = chooseString(override.TenantID, base.TenantID)
	merged.CustomerID = chooseString(override.CustomerID, base.CustomerID)
	merged.AttemptType = chooseString(override.AttemptType, base.AttemptType)
	merged.Plan = chooseString(override.Plan, base.Plan)
	merged.Environment = chooseString(override.Environment, base.Environment)
	merged.TemplateID = chooseString(override.TemplateID, base.TemplateID)
	merged.TraceID = chooseString(override.TraceID, base.TraceID)
	merged.RunID = chooseString(override.RunID, base.RunID)
	merged.ConversationID = chooseString(override.ConversationID, base.ConversationID)
	merged.SpanID = chooseString(override.SpanID, base.SpanID)
	merged.ParentSpanID = chooseString(override.ParentSpanID, base.ParentSpanID)
	merged.StepName = chooseString(override.StepName, base.StepName)
	merged.Outcome = chooseString(override.Outcome, base.Outcome)
	merged.RetryReason = chooseString(override.RetryReason, base.RetryReason)
	merged.FallbackReason = chooseString(override.FallbackReason, base.FallbackReason)
	merged.QualityLabel = chooseString(override.QualityLabel, base.QualityLabel)
	merged.FeedbackScore = chooseFloatPointer(override.FeedbackScore, base.FeedbackScore)
	if override.CaptureContent {
		merged.CaptureContent = true
	}
	if override.EmitLifecycleEvents {
		merged.EmitLifecycleEvents = true
	}
	merged.SchemaVersion = chooseString(override.SchemaVersion, base.SchemaVersion)
	merged.SpanKind = chooseString(override.SpanKind, base.SpanKind)
	merged.ToolName = chooseString(override.ToolName, base.ToolName)
	merged.Provider = chooseString(override.Provider, base.Provider)
	merged.EventType = chooseString(override.EventType, base.EventType)
	merged.Endpoint = chooseString(override.Endpoint, base.Endpoint)
	merged.Model = chooseString(override.Model, base.Model)
	if len(override.PayloadRefs) > 0 {
		merged.PayloadRefs = append([]string{}, override.PayloadRefs...)
	}
	if len(override.PayloadBlocks) > 0 {
		merged.PayloadBlocks = append([]TracePayloadBlock{}, override.PayloadBlocks...)
	}
	if override.Metrics != nil {
		merged.Metrics = override.Metrics
	}
	if override.Decision != nil {
		merged.Decision = override.Decision
	}
	if len(override.Headers) > 0 {
		merged.Headers = cloneHeaders(override.Headers)
	}
	return merged
}

func mergeMetrics(base *TraceMetrics, override *TraceMetrics) *TraceMetrics {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		copied := *override
		return &copied
	}
	if override == nil {
		copied := *base
		return &copied
	}
	merged := *base
	if override.PromptTokens != 0 {
		merged.PromptTokens = override.PromptTokens
	}
	if override.CompletionTokens != 0 {
		merged.CompletionTokens = override.CompletionTokens
	}
	if override.TotalTokens != 0 {
		merged.TotalTokens = override.TotalTokens
	}
	if override.CostUSD != 0 {
		merged.CostUSD = override.CostUSD
	}
	if override.LatencyMs != 0 {
		merged.LatencyMs = override.LatencyMs
	}
	return &merged
}

func mergeDecision(base *TraceDecision, override *TraceDecision) *TraceDecision {
	if base == nil && override == nil {
		return nil
	}
	if base == nil {
		copied := *override
		return &copied
	}
	if override == nil {
		copied := *base
		return &copied
	}
	merged := *base
	merged.RetryReason = chooseString(override.RetryReason, base.RetryReason)
	merged.FallbackReason = chooseString(override.FallbackReason, base.FallbackReason)
	merged.RoutingReason = chooseString(override.RoutingReason, base.RoutingReason)
	merged.Route = chooseString(override.Route, base.Route)
	return &merged
}

func stringifyPayload(payload any) (string, error) {
	switch value := payload.(type) {
	case string:
		return value, nil
	case []byte:
		return string(value), nil
	default:
		body, err := json.Marshal(value)
		if err != nil {
			return "", fmt.Errorf("tokvera: encode payload: %w", err)
		}
		return string(body), nil
	}
}

func chooseString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func chooseFloatPointer(values ...*float64) *float64 {
	for _, value := range values {
		if value != nil {
			copied := *value
			return &copied
		}
	}
	return nil
}

func generateID(prefix string) string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(buffer)
}

func cloneHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(headers))
	for key, value := range headers {
		cloned[key] = value
	}
	return cloned
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func mapStatusToOutcome(status string) string {
	switch status {
	case "success":
		return "success"
	case "failure":
		return "failure"
	default:
		return "in_progress"
	}
}

func decisionRetry(decision *TraceDecision) string {
	if decision == nil {
		return ""
	}
	return decision.RetryReason
}

func decisionFallback(decision *TraceDecision) string {
	if decision == nil {
		return ""
	}
	return decision.FallbackReason
}
