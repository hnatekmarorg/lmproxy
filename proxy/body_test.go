package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hnatekmarorg/lmproxy/config"
)

func TestPrepareRequestBody_NoMerge(t *testing.T) {
	req := httptest.NewRequest("POST", "/test", bytes.NewBuffer([]byte(`{"key":"value"}`)))

	body, err := prepareRequestBody(req, nil, "/v1/chat/completions", 0) // 0 = no limit
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	data, _ := io.ReadAll(body)
	if string(data) != `{"key":"value"}` {
		t.Errorf("Expected body unchanged, got %q", string(data))
	}
}

func TestPrepareRequestBody_MergeBody(t *testing.T) {
	modelConfig := &config.ModelConfig{
		Body: map[string]interface{}{
			"temperature": 0.7,
			"max_tokens":  100,
		},
	}

	req := httptest.NewRequest("POST", "/test", bytes.NewBuffer([]byte(`{"messages":[{"role":"user","content":"test"}]}`)))

	body, err := prepareRequestBody(req, modelConfig, "/v1/chat/completions", 10*1024*1024) // 10MB limit
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(body.(*bytes.Buffer).Bytes(), &result)

	if result["temperature"] != 0.7 {
		t.Errorf("Expected temperature 0.7, got %v", result["temperature"])
	}
	if result["max_tokens"] != 100.0 {
		t.Errorf("Expected max_tokens 100, got %v", result["max_tokens"])
	}
}

func TestPrepareRequestBody_NotInferenceEndpoint(t *testing.T) {
	modelConfig := &config.ModelConfig{
		Body: map[string]interface{}{"temperature": 0.7},
	}

	req := httptest.NewRequest("POST", "/test", bytes.NewBuffer([]byte(`{"key":"value"}`)))

	body, err := prepareRequestBody(req, modelConfig, "/v1/models", 10*1024*1024) // 10MB limit
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	data, _ := io.ReadAll(body)
	if string(data) != `{"key":"value"}` {
		t.Errorf("Expected body unchanged for non-inference endpoint, got %q", string(data))
	}
}

func TestPrepareRequestBody_GETRequest(t *testing.T) {
	modelConfig := &config.ModelConfig{
		Body: map[string]interface{}{"temperature": 0.7},
	}

	req := httptest.NewRequest("GET", "/test", nil)

	body, err := prepareRequestBody(req, modelConfig, "/v1/chat/completions", 10*1024*1024) // 10MB limit
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if body != req.Body {
		t.Errorf("Expected body to be unchanged for GET request")
	}
}

func TestPrepareRequestBody_RequestBodySizeLimit(t *testing.T) {
	modelConfig := &config.ModelConfig{
		Body: map[string]interface{}{"temperature": 0.7},
	}

	// Create a body larger than the 1KB limit
	largeBody := make([]byte, 2*1024) // 2KB
	for i := range largeBody {
		largeBody[i] = 'x'
	}

	req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(largeBody))

	// 1KB limit should cause error
	_, err := prepareRequestBody(req, modelConfig, "/v1/chat/completions", 1024)
	if err == nil {
		t.Fatal("Expected error for body exceeding limit, got nil")
	}
	if !strings.Contains(err.Error(), "request entity too large") && !strings.Contains(err.Error(), "MaxBytes") {
		t.Logf("Note: Error message: %v", err)
	}
}

func TestPrepareRequestBody_NoLimitForPassThrough(t *testing.T) {
	// When no merge is needed, body should pass through without limit
	largeBody := make([]byte, 2*1024) // 2KB
	for i := range largeBody {
		largeBody[i] = 'x'
	}

	req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(largeBody))

	// 1KB limit but no merge needed - should pass through
	body, err := prepareRequestBody(req, nil, "/v1/chat/completions", 1024)
	if err != nil {
		t.Fatalf("Expected no error for pass-through, got %v", err)
	}
	if body != req.Body {
		t.Errorf("Expected body to pass through unchanged")
	}
}
