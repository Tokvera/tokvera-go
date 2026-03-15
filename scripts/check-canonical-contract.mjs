const DEFAULT_BASE_URL = "https://api.tokvera.org";

const EXPECTED_V1 = {
  envelope_version: "v1",
  schema_version: "2026-02-16",
  optional_top_level_fields: ["prompt_hash", "response_hash", "error", "evaluation"],
};

const EXPECTED_V2 = {
  envelope_version: "v2",
  schema_version: "2026-04-01",
  optional_top_level_fields: [
    "prompt_hash",
    "response_hash",
    "error",
    "evaluation",
    "span_kind",
    "tool_name",
    "payload_refs",
    "payload_blocks",
    "metrics",
    "decision",
  ],
  span_kinds: ["model", "tool", "orchestrator", "retrieval", "guardrail"],
  payload_types: ["prompt_input", "tool_input", "tool_output", "model_output", "context", "other"],
  metrics_fields: ["prompt_tokens", "completion_tokens", "total_tokens", "latency_ms", "cost_usd"],
  decision_fields: ["outcome", "retry_reason", "fallback_reason", "routing_reason", "route"],
};

const REQUIRED_TOP_LEVEL_FIELDS = [
  "schema_version",
  "event_type",
  "provider",
  "endpoint",
  "status",
  "timestamp",
  "latency_ms",
  "model",
  "usage",
  "tags",
];
const STATUS_VALUES = ["in_progress", "success", "failure"];
const PROVIDER_CONTRACTS = {
  openai: { event_type: "openai.request", endpoints: ["chat.completions.create", "responses.create"] },
  anthropic: { event_type: "anthropic.request", endpoints: ["messages.create"] },
  gemini: { event_type: "gemini.request", endpoints: ["models.generate_content"] },
  mistral: { event_type: "mistral.request", endpoints: ["chat.complete"] },
  tokvera: { event_type: "tokvera.trace", endpoints: ["manual.trace", "manual.span", "otel.span"] },
};
const USAGE_FIELDS = ["prompt_tokens", "completion_tokens", "total_tokens"];
const ERROR_FIELDS = ["type", "message"];
const ALLOWED_TAG_FIELDS = [
  "feature",
  "tenant_id",
  "customer_id",
  "attempt_type",
  "plan",
  "environment",
  "template_id",
  "trace_id",
  "run_id",
  "conversation_id",
  "span_id",
  "parent_span_id",
  "step_name",
  "outcome",
  "retry_reason",
  "fallback_reason",
  "quality_label",
  "feedback_score",
];
const EVALUATION_FIELDS = ["outcome", "retry_reason", "fallback_reason", "quality_label", "feedback_score"];
const VALIDATION_ERROR_CODES_V1 = [
  "MISSING_FIELD",
  "UNSUPPORTED_VERSION",
  "UNSUPPORTED_EVENT_TYPE",
  "INVALID_SCHEMA",
  "UNKNOWN_TOP_LEVEL_FIELD",
  "UNKNOWN_USAGE_FIELD",
  "UNKNOWN_TAG_FIELD",
  "UNKNOWN_EVALUATION_FIELD",
  "UNKNOWN_ERROR_FIELD",
];
const VALIDATION_ERROR_CODES_V2 = [
  ...VALIDATION_ERROR_CODES_V1,
  "UNKNOWN_METRICS_FIELD",
  "UNKNOWN_DECISION_FIELD",
];

function asSortedSet(values) {
  return [...new Set(values || [])].sort();
}

function assertEqual(actual, expected, label) {
  if (actual !== expected) {
    throw new Error(`${label} mismatch. expected=${expected} actual=${actual}`);
  }
}

function assertSetEqual(actual, expected, label) {
  const actualSorted = asSortedSet(actual);
  const expectedSorted = asSortedSet(expected);
  if (JSON.stringify(actualSorted) !== JSON.stringify(expectedSorted)) {
    throw new Error(`${label} mismatch. expected=${JSON.stringify(expectedSorted)} actual=${JSON.stringify(actualSorted)}`);
  }
}

async function fetchSchema(url) {
  const response = await fetch(url, { method: "GET" });
  if (!response.ok) {
    throw new Error(`Canonical contract request failed with HTTP ${response.status} for ${url}`);
  }
  const payload = await response.json();
  if (!payload?.ok || !payload?.schema || typeof payload.schema !== "object") {
    throw new Error(`Canonical contract response payload format is invalid for ${url}`);
  }
  return payload.schema;
}

function assertCommon(schema, { v2 = false } = {}) {
  assertSetEqual(schema.required_top_level_fields || [], REQUIRED_TOP_LEVEL_FIELDS, "required_top_level_fields");
  assertSetEqual(schema.status_values || [], STATUS_VALUES, "status_values");
  assertSetEqual(schema.usage_fields || [], USAGE_FIELDS, "usage_fields");
  assertSetEqual(schema.error_fields || [], ERROR_FIELDS, "error_fields");
  assertSetEqual(schema.allowed_tag_fields || [], ALLOWED_TAG_FIELDS, "allowed_tag_fields");
  assertSetEqual(schema.evaluation_fields || [], EVALUATION_FIELDS, "evaluation_fields");
  assertSetEqual(schema.validation_error_codes || [], v2 ? VALIDATION_ERROR_CODES_V2 : VALIDATION_ERROR_CODES_V1, "validation_error_codes");
  for (const [provider, expected] of Object.entries(PROVIDER_CONTRACTS)) {
    const actual = schema.provider_contracts?.[provider];
    if (!actual) {
      throw new Error(`provider_contracts.${provider} missing`);
    }
    assertEqual(actual.event_type, expected.event_type, `provider_contracts.${provider}.event_type`);
    assertSetEqual(actual.endpoints || [], expected.endpoints, `provider_contracts.${provider}.endpoints`);
  }
}

function assertV1(schema) {
  assertEqual(schema.envelope_version, EXPECTED_V1.envelope_version, "envelope_version");
  assertEqual(schema.schema_version, EXPECTED_V1.schema_version, "schema_version");
  assertSetEqual(schema.optional_top_level_fields || [], EXPECTED_V1.optional_top_level_fields, "optional_top_level_fields");
  assertCommon(schema, { v2: false });
}

function assertV2(schema) {
  assertEqual(schema.envelope_version, EXPECTED_V2.envelope_version, "envelope_version");
  assertEqual(schema.schema_version, EXPECTED_V2.schema_version, "schema_version");
  assertSetEqual(schema.optional_top_level_fields || [], EXPECTED_V2.optional_top_level_fields, "optional_top_level_fields");
  assertSetEqual(schema.span_kinds || [], EXPECTED_V2.span_kinds, "span_kinds");
  assertSetEqual(schema.payload_types || [], EXPECTED_V2.payload_types, "payload_types");
  assertSetEqual(schema.metrics_fields || [], EXPECTED_V2.metrics_fields, "metrics_fields");
  assertSetEqual(schema.decision_fields || [], EXPECTED_V2.decision_fields, "decision_fields");
  assertCommon(schema, { v2: true });
}

async function main() {
  const baseURL = (process.env.TOKVERA_API_BASE_URL || DEFAULT_BASE_URL).replace(/\/$/, "");
  const v1 = await fetchSchema(`${baseURL}/v1/schema/event-envelope-v1`);
  const v2 = await fetchSchema(`${baseURL}/v1/schema/event-envelope-v2`);
  assertV1(v1);
  assertV2(v2);
  console.log(`tokvera-go canonical contract check passed against ${baseURL}`);
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : error);
  process.exit(1);
});
