package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/hnatekmarorg/lmproxy/config"
)

func (p *Proxy) resolveTargetURL(requestURL *url.URL, requestID string) (*config.Endpoint, *config.ModelConfig, *url.URL) {
	if len(p.endpoints) == 0 {
		return nil, nil, nil
	}

	// Search all endpoints for a matching model path
	for i := range p.endpoints {
		endpoint := &p.endpoints[i]
		modelConfig := p.findModelForPath(endpoint, requestURL.Path)
		if modelConfig != nil {
			targetURL, err := url.Parse(endpoint.Host)
			if err != nil {
				slog.Error("Invalid endpoint URL", "request_id", requestID, "host", endpoint.Host, "error", err)
				continue // Try next endpoint
			}
			slog.Debug("Route resolved", "request_id", requestID, "model", modelConfig.ID, "endpoint", endpoint.Host, "path", requestURL.Path)
			return endpoint, modelConfig, targetURL
		}
	}

	// No matching model found
	slog.Warn("No matching model found for path", "request_id", requestID, "path", requestURL.Path)
	return nil, nil, nil
}

func (p *Proxy) findModelForPath(endpoint *config.Endpoint, path string) *config.ModelConfig {
	for _, model := range endpoint.Models {
		if model.Path != "" && strings.HasPrefix(path, model.Path) {
			slog.Debug("Model selected", "model_id", model.ID, "model_path", model.Path, "request_path", path)
			return &model
		}
	}
	return nil
}

func (p *Proxy) computeTargetPath(requestPath string, modelConfig *config.ModelConfig) string {
	targetPath := requestPath

	if modelConfig != nil && modelConfig.Path != "" {
		targetPath = strings.TrimPrefix(requestPath, modelConfig.Path)

		if targetPath == "" || targetPath == "/" {
			targetPath = "/v1/chat/completions"
		} else if !strings.HasPrefix(targetPath, "/") {
			targetPath = "/" + targetPath
		}
	}

	return targetPath
}

// resolveTargetByModelID attempts to route a request by reading the "model" field
// from the POST body and finding the matching endpoint and model config.
// This is used when path-based routing fails (e.g., for models without a path).
func (p *Proxy) resolveTargetByModelID(r *http.Request, requestID string) (*url.URL, *config.ModelConfig) {
	if r.Method != "POST" {
		return nil, nil
	}

	// Read just enough of the body to extract the model field
	// Limit to a reasonable size to avoid buffering large requests
	bodyBytes := make([]byte, 1024*64) // 64KB should be enough for model field
	n, _ := r.Body.Read(bodyBytes)
	if n == 0 {
		return nil, nil
	}
	bodyBytes = bodyBytes[:n]

	// We need to re-create the body since we read part of it
	r.Body = io.NopCloser(io.MultiReader(
		bytes.NewReader(bodyBytes),
		r.Body,
	))

	// Parse the model field from the body
	var partial struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(bodyBytes, &partial); err != nil || partial.Model == "" {
		return nil, nil
	}

	slog.Debug("Body-based routing", "request_id", requestID, "model", partial.Model)

	// Find the endpoint and model config
	for i := range p.endpoints {
		endpoint := &p.endpoints[i]
		for j := range endpoint.Models {
			if endpoint.Models[j].ID == partial.Model {
				targetURL, err := url.Parse(endpoint.Host)
				if err != nil {
					slog.Error("Invalid endpoint URL", "request_id", requestID, "host", endpoint.Host, "error", err)
					return nil, nil
				}
				slog.Debug("Route resolved by model ID", "request_id", requestID, "model", partial.Model, "endpoint", endpoint.Host)
				return targetURL, &endpoint.Models[j]
			}
		}
	}

	slog.Warn("No matching model found for model ID", "request_id", requestID, "model", partial.Model)
	return nil, nil
}
