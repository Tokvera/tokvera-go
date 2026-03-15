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
		return "openai.request"
	case "anthropic":
		return "anthropic.request"
	case "gemini":
		return "gemini.request"
	case "mistral":
		return "mistral.request"
	default:
		return "tokvera.trace"
	}
}

func defaultProviderEndpoint(provider string) string {
	switch provider {
	case "openai":
		return "responses.create"
	case "anthropic":
		return "messages.create"
	case "gemini":
		return "models.generate_content"
	case "mistral":
		return "chat.complete"
	default:
		return "manual.span"
	}
}
