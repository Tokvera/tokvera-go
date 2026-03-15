package tokvera

import "context"

type ProviderOperation func(context.Context) (ProviderResult, error)

func (tracer *Tracer) TrackOpenAI(
	ctx context.Context,
	parent TraceHandle,
	request ProviderRequest,
	operation ProviderOperation,
) (ProviderResult, error) {
	return tracer.trackProvider(ctx, parent, "openai", request, operation)
}

func (tracer *Tracer) TrackAnthropic(
	ctx context.Context,
	parent TraceHandle,
	request ProviderRequest,
	operation ProviderOperation,
) (ProviderResult, error) {
	return tracer.trackProvider(ctx, parent, "anthropic", request, operation)
}

func (tracer *Tracer) TrackGemini(
	ctx context.Context,
	parent TraceHandle,
	request ProviderRequest,
	operation ProviderOperation,
) (ProviderResult, error) {
	return tracer.trackProvider(ctx, parent, "gemini", request, operation)
}

func (tracer *Tracer) TrackMistral(
	ctx context.Context,
	parent TraceHandle,
	request ProviderRequest,
	operation ProviderOperation,
) (ProviderResult, error) {
	return tracer.trackProvider(ctx, parent, "mistral", request, operation)
}

func (tracer *Tracer) trackProvider(
	ctx context.Context,
	parent TraceHandle,
	provider string,
	request ProviderRequest,
	operation ProviderOperation,
) (ProviderResult, error) {
	child, err := tracer.StartSpan(ctx, parent, TrackOptions{
		Provider:  provider,
		EventType: chooseString(request.EventType, defaultProviderEventType(provider)),
		Endpoint:  chooseString(request.Endpoint, defaultProviderEndpoint(provider)),
		Model:     request.Model,
		StepName:  chooseString(request.StepName, provider+"_call"),
		SpanKind:  chooseString(request.SpanKind, "model"),
		ToolName:  request.ToolName,
		Headers:   cloneHeaders(request.Headers),
	})
	if err != nil {
		return ProviderResult{}, err
	}

	if request.Input != nil && tracer.base.CaptureContent {
		attached, attachErr := tracer.AttachPayload(child, request.Input, "prompt_input")
		if attachErr == nil {
			child = attached
		}
	}

	result, err := operation(ctx)
	if err != nil {
		if failErr := tracer.FailSpan(ctx, child, err, FinishSpanOptions{}); failErr != nil {
			return ProviderResult{}, failErr
		}
		return ProviderResult{}, err
	}

	if result.Output != nil && tracer.base.CaptureContent {
		attached, attachErr := tracer.AttachPayload(child, result.Output, "model_output")
		if attachErr == nil {
			child = attached
		}
	}

	finishOptions := FinishSpanOptions{
		Usage:         result.Usage,
		Outcome:       chooseString(result.Outcome, "success"),
		QualityLabel:  result.QualityLabel,
		FeedbackScore: result.FeedbackScore,
		Metrics:       result.Metrics,
		Decision:      result.Decision,
	}
	if err := tracer.FinishSpan(ctx, child, finishOptions); err != nil {
		return ProviderResult{}, err
	}
	return result, nil
}

func defaultProviderEventType(provider string) string {
	switch provider {
	case "openai":
		return "responses_create"
	case "anthropic":
		return "messages_create"
	case "gemini":
		return "generate_content"
	case "mistral":
		return "chat_complete"
	default:
		return "provider_call"
	}
}

func defaultProviderEndpoint(provider string) string {
	switch provider {
	case "openai":
		return "/v1/responses"
	case "anthropic":
		return "/v1/messages"
	case "gemini":
		return "/v1/models/generateContent"
	case "mistral":
		return "/v1/chat/completions"
	default:
		return "provider"
	}
}
