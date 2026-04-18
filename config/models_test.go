package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `
endpoints:
  - host: https://example.com
    models:
      - id: test-model
        path: /test
        body:
          temperature: 0.7
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(cfg.Endpoints) != 1 {
		t.Fatalf("Expected 1 endpoint, got %d", len(cfg.Endpoints))
	}
	if cfg.Endpoints[0].Host != "https://example.com" {
		t.Errorf("Expected host 'https://example.com', got %q", cfg.Endpoints[0].Host)
	}
}

func TestLoad_DefaultServerConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `
endpoints:
  - host: https://example.com
    models:
      - id: test-model
        path: /test
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default host '0.0.0.0', got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", cfg.Server.Port)
	}
}

func TestLoad_NoEndpoints(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `endpoints: []`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Expected error for empty endpoints, got nil")
	}
}

func TestLoad_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "empty host",
			content: `
endpoints:
  - host: ""
    models:
      - id: test
        path: /test
`,
		},
		{
			name: "empty model id",
			content: `
endpoints:
  - host: https://example.com
    models:
      - id: ""
        path: /test
`,
		},
		{
			name: "empty model path",
			content: `
endpoints:
  - host: https://example.com
    models:
      - id: test
        path: ""
`,
		},
		{
			name: "path without leading slash",
			content: `
endpoints:
  - host: https://example.com
    models:
      - id: test
        path: test
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := Load(configPath)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
		})
	}
}

func TestLoad_DuplicateModelIDs(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `
endpoints:
  - host: https://example.com
    models:
      - id: duplicate
        path: /test1
      - id: duplicate
        path: /test2
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Expected error for duplicate model IDs, got nil")
	}
}
