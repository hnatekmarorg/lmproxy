package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type ModelConfig struct {
	ID                 string                 `yaml:"id"`
	Body               map[string]interface{} `yaml:"body"`
	Path               string                 `yaml:"path"`
	ExtraBody          map[string]interface{} `yaml:"extra_body"`
	ChatTemplateKwargs map[string]interface{} `yaml:"chat_template_kwargs"`
}

type Endpoint struct {
	Host   string        `yaml:"host"`
	Models []ModelConfig `yaml:"models"`
}

type HTTPConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Config struct {
	Server    HTTPConfig `yaml:"server"`
	Endpoints []Endpoint `yaml:"endpoints"`
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

	// Validate endpoints
	seenModelIDs := make(map[string]bool)
	for i, endpoint := range config.Endpoints {
		if endpoint.Host == "" {
			return nil, fmt.Errorf("endpoint %d: host is required", i)
		}
		if _, err := url.Parse(endpoint.Host); err != nil {
			return nil, fmt.Errorf("endpoint %d: invalid host URL %q: %w", i, endpoint.Host, err)
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
			if model.Path == "" {
				return nil, fmt.Errorf("endpoint %d, model %d (%s): path is required", i, j, model.ID)
			}
			if !strings.HasPrefix(model.Path, "/") {
				return nil, fmt.Errorf("endpoint %d, model %d (%s): path must start with /", i, j, model.ID)
			}
		}
	}

	return &config, nil
}
