package main

import (
	"context"
	"log"
	"strconv"
	"time"

	tokvera "github.com/tokvera/tokvera-go"
	"github.com/tokvera/tokvera-go/examples/internal/exampleenv"
)

func main() {
	base := exampleenv.BaseOptions("go_otel_bridge", false)
	bridge := tokvera.NewOTelBridge(base)
	now := time.Now().UTC()
	traceID := "trc_go_otel_" + strconv.FormatInt(now.UnixMilli(), 10)
	spanID := "spn_go_otel_" + strconv.FormatInt(now.UnixMilli(), 10)

	err := bridge.Export(context.Background(), []tokvera.OTelReadableSpan{
		{
			Name:       "llm_call",
			TraceID:    traceID,
			SpanID:     spanID,
			StartTime:  now.Add(-250 * time.Millisecond),
			EndTime:    now,
			StatusCode: "ok",
			Attributes: map[string]any{
				"tokvera.feature":            base.Feature,
				"tokvera.tenant_id":          base.TenantID,
				"tokvera.provider":           "openai",
				"tokvera.event_type":         "openai.request",
				"tokvera.endpoint":           "responses.create",
				"tokvera.step_name":          "otel_bridge_model",
				"gen_ai.request.model":       "gpt-4o-mini",
				"gen_ai.usage.prompt_tokens": int64(21),
				"gen_ai.usage.total_tokens":  int64(34),
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
