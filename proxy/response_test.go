package proxy

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type mockFlusher struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (m *mockFlusher) Flush() {
	m.flushed = true
}

func TestForwardResponseBody_NonSSE(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := io.NopCloser(bytes.NewBuffer([]byte("test response")))

	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       body,
	}

	forwardResponseBody(recorder, resp, "test-request-id", time.Now())

	if recorder.Body.String() != "test response" {
		t.Errorf("Expected 'test response', got %q", recorder.Body.String())
	}
}

func TestForwardResponseBody_SSE(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := io.NopCloser(bytes.NewBuffer([]byte("event: data\n\n")))

	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       body,
	}

	forwardResponseBody(recorder, resp, "test-request-id", time.Now())

	if !strings.Contains(recorder.Body.String(), "event: data") {
		t.Errorf("Expected SSE content, got %q", recorder.Body.String())
	}
}
