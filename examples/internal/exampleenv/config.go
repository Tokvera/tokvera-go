package exampleenv

import (
	"os"
	"strings"

	tokvera "github.com/tokvera/tokvera-go"
)

func BaseOptions(defaultFeature string, lifecycle bool) tokvera.TrackOptions {
	baseURL := env("TOKVERA_BASE_URL", "")
	if baseURL == "" {
		baseURL = normalizeBaseURL(env("TOKVERA_INGEST_URL", tokvera.DefaultBaseURL))
	}

	return tokvera.TrackOptions{
		APIKey:              env("TOKVERA_API_KEY", "tok_live_preview"),
		BaseURL:             baseURL,
		Feature:             env("TOKVERA_FEATURE", defaultFeature),
		TenantID:            env("TOKVERA_TENANT_ID", "tenant_demo"),
		CustomerID:          env("TOKVERA_CUSTOMER_ID", ""),
		Environment:         env("TOKVERA_ENVIRONMENT", "dev"),
		CaptureContent:      envBool("TOKVERA_CAPTURE_CONTENT", true),
		EmitLifecycleEvents: envBool("TOKVERA_EMIT_LIFECYCLE", lifecycle),
	}
}

func normalizeBaseURL(value string) string {
	trimmed := unquote(strings.TrimSpace(value))
	trimmed = strings.TrimRight(trimmed, "/")
	trimmed = strings.TrimSuffix(trimmed, "/v1/events")
	if trimmed == "" {
		return tokvera.DefaultBaseURL
	}
	return trimmed
}

func env(key string, fallback string) string {
	value := unquote(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(unquote(os.Getenv(key))))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func unquote(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.Trim(trimmed, "\"")
	trimmed = strings.Trim(trimmed, "'")
	return strings.TrimSpace(trimmed)
}
