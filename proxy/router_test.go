package proxy

import (
	"testing"

	"github.com/hnatekmarorg/lmproxy/config"
)

func TestFindModelForPath(t *testing.T) {
	endpoint := &config.Endpoint{
		Models: []config.ModelConfig{
			{ID: "model1", Path: "/fast"},
			{ID: "model2", Path: "/slow"},
		},
	}

	p := &Proxy{}

	tests := []struct {
		path     string
		expected string
	}{
		{"/fast/v1/chat", "model1"},
		{"/fast/something", "model1"},
		{"/slow/completions", "model2"},
		{"/unknown", ""},
		{"/", ""},
	}

	for _, tt := range tests {
		model := p.findModelForPath(endpoint, tt.path)
		if tt.expected == "" {
			if model != nil {
				t.Errorf("Path %q: expected nil, got %v", tt.path, model.ID)
			}
		} else {
			if model == nil || model.ID != tt.expected {
				t.Errorf("Path %q: expected %q, got %v", tt.path, tt.expected, model)
			}
		}
	}
}
