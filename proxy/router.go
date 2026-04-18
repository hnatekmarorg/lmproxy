package proxy

import (
	"log/slog"
	"net/url"
	"strings"

	"github.com/hnatekmarorg/lmproxy/config"
)

func (p *Proxy) resolveTargetURL(requestURL *url.URL) (*config.Endpoint, *url.URL) {
	if len(p.endpoints) == 0 {
		return nil, nil
	}

	endpoint := &p.endpoints[0]

	targetURL, err := url.Parse(endpoint.Host)
	if err != nil {
		slog.Error("Invalid endpoint URL", "host", endpoint.Host, "error", err)
		return nil, nil
	}

	return endpoint, targetURL
}

func (p *Proxy) findModelForPath(endpoint *config.Endpoint, path string) *config.ModelConfig {
	for _, model := range endpoint.Models {
		if model.Path != "" && strings.HasPrefix(path, model.Path) {
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
