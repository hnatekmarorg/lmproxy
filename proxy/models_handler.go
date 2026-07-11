package proxy

import (
	"encoding/json"
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

func (p *Proxy) handleListModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	models := p.getAllModels()
	resp := ListModelsResponse{
		Object: "list",
		Data:   models,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode models response", "error", err)
	}
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
