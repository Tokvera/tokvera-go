package tokvera

import "time"

const (
	DefaultBaseURL       = "https://api.tokvera.org"
	TraceSchemaVersionV1 = "2026-02-16"
	TraceSchemaVersionV2 = "2026-04-01"
)

type Usage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

type EventError struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	Code    string `json:"code,omitempty"`
}

type TracePayloadBlock struct {
	PayloadType string `json:"payload_type"`
	Content     string `json:"content"`
}

type TraceMetrics struct {
	PromptTokens     int     `json:"prompt_tokens,omitempty"`
	CompletionTokens int     `json:"completion_tokens,omitempty"`
	TotalTokens      int     `json:"total_tokens,omitempty"`
	CostUSD          float64 `json:"cost_usd,omitempty"`
	LatencyMs        int64   `json:"latency_ms,omitempty"`
}

type TraceDecision struct {
	RetryReason    string `json:"retry_reason,omitempty"`
	FallbackReason string `json:"fallback_reason,omitempty"`
	RoutingReason  string `json:"routing_reason,omitempty"`
	Route          string `json:"route,omitempty"`
}

type TrackOptions struct {
	APIKey              string              `json:"-"`
	BaseURL             string              `json:"-"`
	Feature             string              `json:"-"`
	TenantID            string              `json:"-"`
	CustomerID          string              `json:"-"`
	AttemptType         string              `json:"-"`
	Plan                string              `json:"-"`
	Environment         string              `json:"-"`
	TemplateID          string              `json:"-"`
	TraceID             string              `json:"-"`
	RunID               string              `json:"-"`
	ConversationID      string              `json:"-"`
	SpanID              string              `json:"-"`
	ParentSpanID        string              `json:"-"`
	StepName            string              `json:"-"`
	Outcome             string              `json:"-"`
	RetryReason         string              `json:"-"`
	FallbackReason      string              `json:"-"`
	QualityLabel        string              `json:"-"`
	FeedbackScore       *float64            `json:"-"`
	CaptureContent      bool                `json:"-"`
	EmitLifecycleEvents bool                `json:"-"`
	SchemaVersion       string              `json:"-"`
	SpanKind            string              `json:"-"`
	ToolName            string              `json:"-"`
	Provider            string              `json:"-"`
	EventType           string              `json:"-"`
	Endpoint            string              `json:"-"`
	Model               string              `json:"-"`
	PayloadRefs         []string            `json:"-"`
	PayloadBlocks       []TracePayloadBlock `json:"-"`
	Metrics             *TraceMetrics       `json:"-"`
	Decision            *TraceDecision      `json:"-"`
	Headers             map[string]string   `json:"-"`
}

type Event struct {
	Provider       string              `json:"provider"`
	EventType      string              `json:"event_type"`
	Endpoint       string              `json:"endpoint"`
	Model          string              `json:"model,omitempty"`
	Status         string              `json:"status"`
	Timestamp      time.Time           `json:"timestamp"`
	Feature        string              `json:"feature"`
	TenantID       string              `json:"tenant_id"`
	CustomerID     string              `json:"customer_id,omitempty"`
	AttemptType    string              `json:"attempt_type,omitempty"`
	Plan           string              `json:"plan,omitempty"`
	Environment    string              `json:"environment,omitempty"`
	TemplateID     string              `json:"template_id,omitempty"`
	TraceID        string              `json:"trace_id,omitempty"`
	RunID          string              `json:"run_id,omitempty"`
	ConversationID string              `json:"conversation_id,omitempty"`
	SpanID         string              `json:"span_id,omitempty"`
	ParentSpanID   string              `json:"parent_span_id,omitempty"`
	StepName       string              `json:"step_name,omitempty"`
	Outcome        string              `json:"outcome,omitempty"`
	RetryReason    string              `json:"retry_reason,omitempty"`
	FallbackReason string              `json:"fallback_reason,omitempty"`
	QualityLabel   string              `json:"quality_label,omitempty"`
	FeedbackScore  *float64            `json:"feedback_score,omitempty"`
	SchemaVersion  string              `json:"schema_version,omitempty"`
	SpanKind       string              `json:"span_kind,omitempty"`
	ToolName       string              `json:"tool_name,omitempty"`
	PayloadRefs    []string            `json:"payload_refs,omitempty"`
	PayloadBlocks  []TracePayloadBlock `json:"payload_blocks,omitempty"`
	Metrics        *TraceMetrics       `json:"metrics,omitempty"`
	Decision       *TraceDecision      `json:"decision,omitempty"`
	Usage          Usage               `json:"usage"`
	LatencyMs      int64               `json:"latency_ms,omitempty"`
	Error          *EventError         `json:"error,omitempty"`
}

type TraceHandle struct {
	TraceID      string
	RunID        string
	SpanID       string
	ParentSpanID string
	StartedAt    time.Time
	Provider     string
	EventType    string
	Endpoint     string
	Model        string
	Options      TrackOptions
}

type FinishSpanOptions struct {
	Usage         Usage
	Outcome       string
	QualityLabel  string
	FeedbackScore *float64
	PayloadBlocks []TracePayloadBlock
	Metrics       *TraceMetrics
	Decision      *TraceDecision
	Error         *EventError
}

type ProviderRequest struct {
	Model     string
	EventType string
	Endpoint  string
	StepName  string
	SpanKind  string
	Input     any
	ToolName  string
	Headers   map[string]string
}

type ProviderResult struct {
	Model         string
	Output        any
	Usage         Usage
	Outcome       string
	QualityLabel  string
	FeedbackScore *float64
	Metrics       *TraceMetrics
	Decision      *TraceDecision
}

type OTelReadableSpan struct {
	Name               string
	TraceID            string
	SpanID             string
	ParentSpanID       string
	StartTime          time.Time
	EndTime            time.Time
	StatusCode         string
	StatusDescription  string
	Attributes         map[string]any
	ResourceAttributes map[string]any
}
