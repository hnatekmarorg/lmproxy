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
			{Host: "https://example.com"},
		},
	}

	endpoint, targetURL := p.resolveTargetURL(&url.URL{})
	if endpoint == nil || targetURL == nil {
		t.Fatal("Expected non-nil endpoint and URL")
	}
	if targetURL.Host != "example.com" {
		t.Errorf("Expected host 'example.com', got %q", targetURL.Host)
	}
}

func TestResolveTargetURL_EmptyEndpoints(t *testing.T) {
	p := &Proxy{}

	endpoint, targetURL := p.resolveTargetURL(&url.URL{})
	if endpoint != nil || targetURL != nil {
		t.Errorf("Expected nil for empty endpoints, got endpoint=%v, URL=%v", endpoint, targetURL)
	}
}
