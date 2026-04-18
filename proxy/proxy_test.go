package proxy

import (
	"testing"

	"github.com/hnatekmarorg/lmproxy/config"
)

func TestNewProxy(t *testing.T) {
	cfg := &config.Config{
		Endpoints: []config.Endpoint{
			{
				Host: "https://example.com",
				Models: []config.ModelConfig{
					{ID: "test", Path: "/test"},
				},
			},
		},
	}

	p := NewProxy(cfg)
	if p == nil {
		t.Fatal("Expected non-nil proxy")
	}
	if len(p.endpoints) != 1 {
		t.Errorf("Expected 1 endpoint, got %d", len(p.endpoints))
	}
}
