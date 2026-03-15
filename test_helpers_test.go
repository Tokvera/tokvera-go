package tokvera

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type ingestRecorder struct {
	server *httptest.Server
	mu     sync.Mutex
	events []Event
	auth   []string
}

func newIngestRecorder(t *testing.T) *ingestRecorder {
	t.Helper()
	recorder := &ingestRecorder{}
	recorder.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		var event Event
		if err := json.NewDecoder(request.Body).Decode(&event); err != nil {
			t.Fatalf("decode event: %v", err)
		}
		recorder.mu.Lock()
		recorder.events = append(recorder.events, event)
		recorder.auth = append(recorder.auth, request.Header.Get("Authorization"))
		recorder.mu.Unlock()
		writer.WriteHeader(http.StatusAccepted)
	}))
	return recorder
}

func (recorder *ingestRecorder) Close() {
	recorder.server.Close()
}

func (recorder *ingestRecorder) URL() string {
	return recorder.server.URL
}

func (recorder *ingestRecorder) Events() []Event {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	events := make([]Event, len(recorder.events))
	copy(events, recorder.events)
	return events
}
