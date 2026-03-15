package tokvera

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout    = 2 * time.Second
	defaultMaxRetries = 2
	defaultRetryDelay = 250 * time.Millisecond
)

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	maxRetries int
	retryDelay time.Duration
}

type ClientOption func(*Client)

func WithBaseURL(baseURL string) ClientOption {
	return func(client *Client) {
		if strings.TrimSpace(baseURL) != "" {
			client.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(client *Client) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

func WithRetry(maxRetries int, retryDelay time.Duration) ClientOption {
	return func(client *Client) {
		if maxRetries >= 0 {
			client.maxRetries = maxRetries
		}
		if retryDelay > 0 {
			client.retryDelay = retryDelay
		}
	}
}

func NewClient(apiKey string, options ...ClientOption) *Client {
	client := &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		maxRetries: defaultMaxRetries,
		retryDelay: defaultRetryDelay,
	}
	for _, option := range options {
		option(client)
	}
	return client
}

func (client *Client) IngestEvent(ctx context.Context, event Event) error {
	if strings.TrimSpace(client.apiKey) == "" {
		return fmt.Errorf("tokvera: api key is required")
	}
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("tokvera: marshal event: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= client.maxRetries; attempt++ {
		request, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			client.baseURL+"/v1/events",
			bytes.NewReader(body),
		)
		if err != nil {
			return fmt.Errorf("tokvera: build request: %w", err)
		}
		request.Header.Set("Authorization", "Bearer "+client.apiKey)
		request.Header.Set("Content-Type", "application/json")

		response, err := client.httpClient.Do(request)
		if err != nil {
			lastErr = fmt.Errorf("tokvera: ingest request failed: %w", err)
		} else {
			_ = response.Body.Close()
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				return nil
			}
			lastErr = fmt.Errorf("tokvera: ingest failed with status %d", response.StatusCode)
			if response.StatusCode < 500 && response.StatusCode != http.StatusTooManyRequests {
				return lastErr
			}
		}

		if attempt < client.maxRetries {
			time.Sleep(client.retryDelay)
		}
	}
	return lastErr
}
