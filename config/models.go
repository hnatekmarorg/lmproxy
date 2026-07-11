package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// PermissionConfig mirrors the OpenAI/vLLM permission object for model responses.
type PermissionConfig struct {
	ID                 string `yaml:"id" json:"id"`
	Object             string `yaml:"object" json:"object"`
	Created            int64  `yaml:"created" json:"created"`
	AllowCreateEngine  bool   `yaml:"allow_create_engine" json:"allow_create_engine"`
	AllowSampling      bool   `yaml:"allow_sampling" json:"allow_sampling"`
	AllowLogprobs      bool   `yaml:"allow_logprobs" json:"allow_logprobs"`
	AllowSearchIndices bool   `yaml:"allow_search_indices" json:"allow_search_indices"`
	AllowView          bool   `yaml:"allow_view" json:"allow_view"`
	AllowFineTuning    bool   `yaml:"allow_fine_tuning" json:"allow_fine_tuning"`
	Organization       string `yaml:"organization" json:"organization"`
	Group              *string `yaml:"group" json:"group"`
	IsBlocking         bool   `yaml:"is_blocking" json:"is_blocking"`
}

type ModelConfig struct {
	ID                 string                 `yaml:"id"`
	Body               map[string]interface{} `yaml:"body"`
	Path               string                 `yaml:"path"`
	ExtraBody          map[string]interface{} `yaml:"extra_body"`
	ChatTemplateKwargs map[string]interface{} `yaml:"chat_template_kwargs"`
	MaxModelLen        int64                  `yaml:"max_model_len,omitempty" json:"max_model_len,omitempty"`
	Root               string                 `yaml:"root,omitempty" json:"root,omitempty"`
	Parent             *string                `yaml:"parent,omitempty" json:"parent,omitempty"`
	Permission         []PermissionConfig     `yaml:"permission,omitempty" json:"permission,omitempty"`
}

type Endpoint struct {
	Host   string        `yaml:"host"`
	Models []ModelConfig `yaml:"models"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type HTTPConfig struct {
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	MaxRequestBodySize int    `yaml:"max_request_body_size"`
	Timeout            int    `yaml:"timeout"`
	ReachableOnly      *bool  `yaml:"reachable_only,omitempty"`
}

type Config struct {
	Server    HTTPConfig    `yaml:"server"`
	Logging   LoggingConfig `yaml:"logging"`
	Models    []ModelConfig `yaml:"models,omitempty"`
	Endpoints []Endpoint    `yaml:"endpoints"`
}

func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if len(config.Endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints configured")
	}

	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}

	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}

	if config.Server.MaxRequestBodySize == 0 {
		config.Server.MaxRequestBodySize = 100 * 1024 * 1024
	}

	if config.Server.Timeout == 0 {
		config.Server.Timeout = 3600
	}

	// Default reachable_only to true so /v1/models enriches with upstream metadata by default
	if config.Server.ReachableOnly == nil {
		defaultTrue := true
		config.Server.ReachableOnly = &defaultTrue
	}

	// Default logging config
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "text"
	}

	// Validate endpoints
	seenModelIDs := make(map[string]bool)
	for i, endpoint := range config.Endpoints {
		if endpoint.Host == "" {
			return nil, fmt.Errorf("endpoint %d: host is required", i)
		}
		parsedURL, err := url.Parse(endpoint.Host)
		if err != nil {
			return nil, fmt.Errorf("endpoint %d: invalid host URL %q: %w", i, endpoint.Host, err)
		}
		scheme := strings.ToLower(parsedURL.Scheme)
		if scheme != "http" && scheme != "https" {
			return nil, fmt.Errorf("endpoint %d: host URL must use http:// or https:// scheme, got %q", i, parsedURL.Scheme)
		}
		if len(endpoint.Models) == 0 {
			return nil, fmt.Errorf("endpoint %d: at least one model is required", i)
		}
		for j, model := range endpoint.Models {
			if model.ID == "" {
				return nil, fmt.Errorf("endpoint %d, model %d: id is required", i, j)
			}
			if seenModelIDs[model.ID] {
				return nil, fmt.Errorf("duplicate model id %q", model.ID)
			}
			seenModelIDs[model.ID] = true
			if model.Path != "" && !strings.HasPrefix(model.Path, "/") {
				return nil, fmt.Errorf("endpoint %d, model %d (%s): path must start with /", i, j, model.ID)
			}
		}
	}

	// Validate top-level models, but don't require path
	for i, model := range config.Models {
		if model.ID == "" {
			return nil, fmt.Errorf("top-level model %d: id is required", i)
		}
		if seenModelIDs[model.ID] {
			return nil, fmt.Errorf("duplicate model id %q", model.ID)
		}
		seenModelIDs[model.ID] = true
		if model.Path != "" && !strings.HasPrefix(model.Path, "/") {
			return nil, fmt.Errorf("top-level model %d (%s): path must start with /", i, model.ID)
		}
	}

	return &config, nil
}

// AllEndpointModels returns all models from all endpoints, deduplicated by ID.
// When a model ID appears in multiple endpoints, only the first occurrence is returned.
func (c *Config) AllEndpointModels() []ModelConfig {
	seen := make(map[string]bool)
	var result []ModelConfig
	for _, endpoint := range c.Endpoints {
		for _, model := range endpoint.Models {
			if !seen[model.ID] {
				seen[model.ID] = true
				result = append(result, model)
			}
		}
	}
	return result
}

// EndpointForModel returns the first endpoint that contains a model with the given ID.
func (c *Config) EndpointForModel(modelID string) *Endpoint {
	for i := range c.Endpoints {
		for _, model := range c.Endpoints[i].Models {
			if model.ID == modelID {
				return &c.Endpoints[i]
			}
		}
	}
	return nil
}
