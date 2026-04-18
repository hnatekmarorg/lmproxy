package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/hnatekmarorg/lmproxy/config"
)

func TestPrepareRequestBody_NoMerge(t *testing.T) {
	req := httptest.NewRequest("POST", "/test", bytes.NewBuffer([]byte(`{"key":"value"}`)))

	body, err := prepareRequestBody(req, nil, "/v1/chat/completions")
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

	body, err := prepareRequestBody(req, modelConfig, "/v1/chat/completions")
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

	body, err := prepareRequestBody(req, modelConfig, "/v1/models")
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

	body, err := prepareRequestBody(req, modelConfig, "/v1/chat/completions")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if body != req.Body {
		t.Errorf("Expected body to be unchanged for GET request")
	}
}
