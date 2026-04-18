package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/hnatekmarorg/lmproxy/config"
	"github.com/hnatekmarorg/lmproxy/util"
)

func prepareRequestBody(r *http.Request, modelConfig *config.ModelConfig, targetPath string, maxBodySize int, requestID string) (io.Reader, error) {
	isInferenceEndpoint := strings.HasSuffix(targetPath, "/completions") || strings.HasSuffix(targetPath, "/generate")

	if modelConfig == nil || r.Method != "POST" || !isInferenceEndpoint {
		return r.Body, nil
	}

	if len(modelConfig.Body) == 0 && len(modelConfig.ExtraBody) == 0 && len(modelConfig.ChatTemplateKwargs) == 0 {
		return r.Body, nil
	}

	// Only apply size limit when we need to read/merge the body
	if maxBodySize > 0 {
		r.Body = http.MaxBytesReader(nil, r.Body, int64(maxBodySize))
	}

	var requestBody map[string]interface{}
	originalBodySize := 0
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("Failed to read request body", "request_id", requestID, "error", err)
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		originalBodySize = len(bodyBytes)
		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
				slog.Error("Failed to parse request body as JSON", "request_id", requestID, "error", err, "body_size", len(bodyBytes))
				return nil, fmt.Errorf("invalid JSON in request body: %w", err)
			}
		} else {
			// Handle whitespace-only or empty body
			trimmed := strings.TrimSpace(string(bodyBytes))
			if len(trimmed) > 0 {
				slog.Error("Failed to parse request body as JSON", "request_id", requestID, "error", "whitespace-only body")
				return nil, fmt.Errorf("request body contains only whitespace")
			}
		}
	}

	if requestBody == nil {
		requestBody = make(map[string]interface{})
	}

	slog.Debug("Merging request body", "request_id", requestID, "original_size", originalBodySize, "has_body", len(modelConfig.Body) > 0, "has_extra_body", len(modelConfig.ExtraBody) > 0, "has_chat_template_kwargs", len(modelConfig.ChatTemplateKwargs) > 0)

	util.MergeMap(requestBody, modelConfig.Body, "")
	util.MergeMap(requestBody, modelConfig.ExtraBody, "extra_body")
	util.MergeMap(requestBody, modelConfig.ChatTemplateKwargs, "chat_template_kwargs")

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		slog.Error("Failed to marshal request body", "request_id", requestID, "error", err)
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	slog.Debug("Request body processed", "request_id", requestID, "original_size", originalBodySize, "processed_size", len(bodyBytes), "modified", originalBodySize != len(bodyBytes))
	return bytes.NewBuffer(bodyBytes), nil
}

func buildProxyRequest(r *http.Request, targetURL *url.URL, body io.Reader) (*http.Request, error) {
	proxyReq, err := http.NewRequest(r.Method, targetURL.String(), body)
	if err != nil {
		return nil, err
	}

	proxyReq.Host = targetURL.Host
	return proxyReq, nil
}
