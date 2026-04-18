package proxy

import (
	"net/url"
	"testing"

	"github.com/hnatekmarorg/lmproxy/config"
)

func TestComputeTargetPath(t *testing.T) {
	p := &Proxy{}

	tests := []struct {
		path      string
		modelPath string
		expected  string
	}{
		{"/fast/v1/chat", "/fast", "/v1/chat"},
		{"/fast", "/fast", "/v1/chat/completions"},
		{"/fast/", "/fast", "/v1/chat/completions"},
		{"/fast/custom", "/fast", "/custom"},
		{"", "", ""},
	}

	for _, tt := range tests {
		var modelConfig *config.ModelConfig
		if tt.modelPath != "" {
			modelConfig = &config.ModelConfig{Path: tt.modelPath}
		}
		result := p.computeTargetPath(tt.path, modelConfig)
		if result != tt.expected {
			t.Errorf("Path %q with model %q: expected %q, got %q", tt.path, tt.modelPath, tt.expected, result)
		}
	}
}

func TestResolveTargetURL(t *testing.T) {
	p := &Proxy{
		endpoints: []config.Endpoint{
			{
				Host: "https://example.com",
				Models: []config.ModelConfig{
					{ID: "test-model", Path: "/test"},
				},
			},
		},
	}

	// Test with matching path
	endpoint, modelConfig, targetURL := p.resolveTargetURL(&url.URL{Path: "/test/v1/chat"})
	if endpoint == nil || modelConfig == nil || targetURL == nil {
		t.Fatal("Expected endpoint, modelConfig, and URL")
	}
	if targetURL.Host != "example.com" {
		t.Errorf("Expected host 'example.com', got %q", targetURL.Host)
	}
	if modelConfig.ID != "test-model" {
		t.Errorf("Expected model 'test-model', got %q", modelConfig.ID)
	}
}

func TestResolveTargetURL_EmptyEndpoints(t *testing.T) {
	p := &Proxy{}

	endpoint, modelConfig, targetURL := p.resolveTargetURL(&url.URL{})
	if endpoint != nil || modelConfig != nil || targetURL != nil {
		t.Errorf("Expected nil for empty endpoints, got endpoint=%v, modelConfig=%v, URL=%v", endpoint, modelConfig, targetURL)
	}
}

func TestResolveTargetURL_MultipleEndpoints(t *testing.T) {
	p := &Proxy{
		endpoints: []config.Endpoint{
			{
				Host: "https://backend1.com",
				Models: []config.ModelConfig{
					{ID: "model1", Path: "/fast"},
				},
			},
			{
				Host: "https://backend2.com",
				Models: []config.ModelConfig{
					{ID: "model2", Path: "/thinking"},
				},
			},
		},
	}

	// Test routing to first endpoint
	endpoint, modelConfig, targetURL := p.resolveTargetURL(&url.URL{Path: "/fast/v1/chat"})
	if endpoint == nil || modelConfig == nil || targetURL == nil {
		t.Fatal("Expected endpoint, modelConfig, and URL for /fast path")
	}
	if targetURL.Host != "backend1.com" {
		t.Errorf("Expected backend1.com, got %q", targetURL.Host)
	}
	if modelConfig.ID != "model1" {
		t.Errorf("Expected model1, got %q", modelConfig.ID)
	}

	// Test routing to second endpoint
	endpoint, modelConfig, targetURL = p.resolveTargetURL(&url.URL{Path: "/thinking/v1/chat"})
	if endpoint == nil || modelConfig == nil || targetURL == nil {
		t.Fatal("Expected endpoint, modelConfig, and URL for /thinking path")
	}
	if targetURL.Host != "backend2.com" {
		t.Errorf("Expected backend2.com, got %q", targetURL.Host)
	}
	if modelConfig.ID != "model2" {
		t.Errorf("Expected model2, got %q", modelConfig.ID)
	}

	// Test no matching path
	endpoint, modelConfig, targetURL = p.resolveTargetURL(&url.URL{Path: "/unknown"})
	if endpoint != nil || modelConfig != nil || targetURL != nil {
		t.Error("Expected nil for unknown path")
	}
}
