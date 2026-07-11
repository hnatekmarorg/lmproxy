package proxy

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/hnatekmarorg/lmproxy/config"
)

// PermissionResponse mirrors the OpenAI/vLLM permission object in model responses.
type PermissionResponse struct {
	ID                 string  `json:"id"`
	Object             string  `json:"object"`
	Created            int64   `json:"created"`
	AllowCreateEngine  bool    `json:"allow_create_engine"`
	AllowSampling      bool    `json:"allow_sampling"`
	AllowLogprobs      bool    `json:"allow_logprobs"`
	AllowSearchIndices bool    `json:"allow_search_indices"`
	AllowView          bool    `json:"allow_view"`
	AllowFineTuning    bool    `json:"allow_fine_tuning"`
	Organization       string  `json:"organization"`
	Group              *string `json:"group"`
	IsBlocking         bool    `json:"is_blocking"`
}

// ModelResponse represents a single model in the OpenAI /v1/models response.
type ModelResponse struct {
	ID          string               `json:"id"`
	Object      string               `json:"object"`
	Created     int64                `json:"created"`
	OwnedBy     string               `json:"owned_by"`
	Root        string               `json:"root,omitempty"`
	Parent      *string              `json:"parent,omitempty"`
	MaxModelLen int64                `json:"max_model_len,omitempty"`
	Permission  []PermissionResponse `json:"permission,omitempty"`
}

// ListModelsResponse is the OpenAI-compatible response for GET /v1/models.
type ListModelsResponse struct {
	Object string          `json:"object"`
	Data   []ModelResponse `json:"data"`
}

// upstreamListModelsResponse is used to parse upstream /v1/models responses.
type upstreamListModelsResponse struct {
	Object string           `json:"object"`
	Data   []upstreamModel  `json:"data"`
}

type upstreamModel struct {
	ID string `json:"id"`
}

func (p *Proxy) handleListModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var models []ModelResponse
	if p.reachableOnly {
		models = p.getReachableModels()
	} else {
		models = p.getAllModels()
	}

	resp := ListModelsResponse{
		Object: "list",
		Data:   models,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode models response", "error", err)
	}
}

// getReachableModels queries each upstream's /v1/models endpoint in real-time
// and returns only proxy models that are actually available on their respective upstream.
func (p *Proxy) getReachableModels() []ModelResponse {
	// Collect reachable model names per endpoint host
	upstreamModels := p.queryAllUpstreams()

	// Build a set of reachable proxy model identifiers (body.model or model ID)
	reachableSet := make(map[string]bool)
	for _, models := range upstreamModels {
		for _, m := range models {
			reachableSet[m] = true
		}
	}

	// Filter configured models by reachability
	seen := make(map[string]bool)
	var result []ModelResponse
	now := time.Now().Unix()

	for _, model := range p.topLevelModels {
		if !seen[model.ID] && p.isModelReachable(model, reachableSet) {
			seen[model.ID] = true
			result = append(result, modelConfigToResponse(model, now))
		}
	}

	for _, endpoint := range p.endpoints {
		for _, model := range endpoint.Models {
			if !seen[model.ID] && p.isModelReachable(model, reachableSet) {
				seen[model.ID] = true
				result = append(result, modelConfigToResponse(model, now))
			}
		}
	}

	return result
}

// isModelReachable checks if a proxy model is reachable on any upstream.
// Matches on body["model"] first, falls back to model ID.
func (p *Proxy) isModelReachable(model config.ModelConfig, reachableSet map[string]bool) bool {
	// Try body.model first
	if model.Body != nil {
		if modelVal, ok := model.Body["model"]; ok {
			if modelStr, ok := modelVal.(string); ok && modelStr != "" {
				if reachableSet[modelStr] {
					return true
				}
			}
		}
	}
	// Fall back to model ID
	return reachableSet[model.ID]
}

// queryAllUpstreams queries each endpoint's /v1/models endpoint and returns
// a map of host -> model IDs available on that upstream.
func (p *Proxy) queryAllUpstreams() map[string][]string {
	result := make(map[string][]string)

	for _, endpoint := range p.endpoints {
		modelsURL := endpoint.Host + "/v1/models"
		req, err := http.NewRequest(http.MethodGet, modelsURL, nil)
		if err != nil {
			slog.Warn("Failed to create upstream models request", "host", endpoint.Host, "error", err)
			continue
		}
		req.Header.Set("Accept", "application/json")

		resp, err := p.client.Do(req)
		if err != nil {
			slog.Warn("Failed to query upstream models", "host", endpoint.Host, "error", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			slog.Warn("Upstream returned non-200 for /v1/models", "host", endpoint.Host, "status", resp.StatusCode)
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB limit
		if err != nil {
			slog.Warn("Failed to read upstream models response", "host", endpoint.Host, "error", err)
			continue
		}

		var upstreamResp upstreamListModelsResponse
		if err := json.Unmarshal(body, &upstreamResp); err != nil {
			slog.Warn("Failed to parse upstream models response", "host", endpoint.Host, "error", err)
			continue
		}

		var modelIDs []string
		for _, m := range upstreamResp.Data {
			modelIDs = append(modelIDs, m.ID)
		}
		result[endpoint.Host] = modelIDs
		slog.Debug("Queried upstream models", "host", endpoint.Host, "model_count", len(modelIDs))
	}

	return result
}

func (p *Proxy) getAllModels() []ModelResponse {
	// Collect models from top-level config first, then from endpoints
	seen := make(map[string]bool)
	var result []ModelResponse

	now := time.Now().Unix()

	for _, model := range p.topLevelModels {
		if !seen[model.ID] {
			seen[model.ID] = true
			result = append(result, modelConfigToResponse(model, now))
		}
	}

	for _, endpoint := range p.endpoints {
		for _, model := range endpoint.Models {
			if !seen[model.ID] {
				seen[model.ID] = true
				result = append(result, modelConfigToResponse(model, now))
			}
		}
	}

	return result
}

func modelConfigToResponse(model config.ModelConfig, now int64) ModelResponse {
	resp := ModelResponse{
		ID:      model.ID,
		Object:  "model",
		Created: now,
		OwnedBy: "proxy",
	}

	// Mirror vLLM response fields if configured
	if model.MaxModelLen > 0 {
		resp.MaxModelLen = model.MaxModelLen
	}
	if model.Root != "" {
		resp.Root = model.Root
	}
	if model.Parent != nil {
		resp.Parent = model.Parent
	}
	if len(model.Permission) > 0 {
		resp.Permission = make([]PermissionResponse, len(model.Permission))
		for i, p := range model.Permission {
			resp.Permission[i] = PermissionResponse{
				ID:                 p.ID,
				Object:             p.Object,
				Created:            p.Created,
				AllowCreateEngine:  p.AllowCreateEngine,
				AllowSampling:      p.AllowSampling,
				AllowLogprobs:      p.AllowLogprobs,
				AllowSearchIndices: p.AllowSearchIndices,
				AllowView:          p.AllowView,
				AllowFineTuning:    p.AllowFineTuning,
				Organization:       p.Organization,
				Group:              p.Group,
				IsBlocking:         p.IsBlocking,
			}
		}
	}

	return resp
}

// findModelByID searches all endpoint models for the given model ID.
// This is used for body-based routing when path-based routing fails.
func (p *Proxy) findModelByID(modelID string) (*config.Endpoint, *config.ModelConfig) {
	// Check top-level models first (if they have a path)
	for i := range p.endpoints {
		endpoint := &p.endpoints[i]
		for j, model := range endpoint.Models {
			if model.ID == modelID {
				return endpoint, &endpoint.Models[j]
			}
		}
	}
	return nil, nil
}
