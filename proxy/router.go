package proxy

import (
	"log/slog"
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
