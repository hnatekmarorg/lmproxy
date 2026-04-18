package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestLoad_CustomMaxRequestBodySize(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `
server:
  host: 0.0.0.0
  port: 9090
  max_request_body_size: 52428800  # 50MB

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
		t.Fatalf("Expected no error, got %v", err)
	}
	if cfg.Server.MaxRequestBodySize != 50*1024*1024 {
		t.Errorf("Expected MaxRequestBodySize 52428800, got %d", cfg.Server.MaxRequestBodySize)
	}
}

func TestLoad_DefaultMaxRequestBodySize(t *testing.T) {
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
		t.Fatalf("Expected no error, got %v", err)
	}
	// Should default to 100MB
	if cfg.Server.MaxRequestBodySize != 100*1024*1024 {
		t.Errorf("Expected default MaxRequestBodySize 104857600, got %d", cfg.Server.MaxRequestBodySize)
	}
}

func TestLoad_CustomTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `
server:
  host: 0.0.0.0
  port: 9090
  timeout: 7200  # 2 hours

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
		t.Fatalf("Expected no error, got %v", err)
	}
	if cfg.Server.Timeout != 7200 {
		t.Errorf("Expected Timeout 7200, got %d", cfg.Server.Timeout)
	}
}

func TestLoad_DefaultTimeout(t *testing.T) {
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
		t.Fatalf("Expected no error, got %v", err)
	}
	// Should default to 1 hour (3600 seconds)
	if cfg.Server.Timeout != 3600 {
		t.Errorf("Expected default Timeout 3600, got %d", cfg.Server.Timeout)
	}
}

func TestLoad_InvalidURLScheme(t *testing.T) {
	tests := []struct {
		name    string
		scheme  string
	}{
		{"file scheme", "file:///etc/passwd"},
		{"gopher scheme", "gopher://localhost:6379"},
		{"ftp scheme", "ftp://example.com"},
		{"no scheme", "example.com"},
		{"malformed", "://invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			content := fmt.Sprintf(`
endpoints:
  - host: %s
    models:
      - id: test
        path: /test
`, tt.scheme)

			if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := Load(configPath)
			if err == nil {
				t.Fatal("Expected error for invalid URL scheme, got nil")
			}
			// Either parsing error or scheme validation error is acceptable
			if !strings.Contains(err.Error(), "must use http:// or https://") && !strings.Contains(err.Error(), "invalid host URL") {
				t.Errorf("Expected scheme validation or parse error, got: %v", err)
			}
		})
	}
}

func TestLoad_ValidHTTPAndHTTPS(t *testing.T) {
	tests := []struct {
		name string
		host string
	}{
		{"http", "http://example.com"},
		{"https", "https://example.com"},
		{"http with port", "http://localhost:8080"},
		{"https with port", "https://api.example.com:443"},
		{"HTTP uppercase", "HTTP://example.com"},
		{"HTTPS uppercase", "HTTPS://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			content := fmt.Sprintf(`
endpoints:
  - host: %s
    models:
      - id: test
        path: /test
`, tt.host)

			if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := Load(configPath)
			if err != nil {
				t.Errorf("Expected no error for valid scheme %q, got %v", tt.host, err)
			}
		})
	}
}
