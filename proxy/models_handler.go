package proxy

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/hnatekmarorg/lmproxy/config"
)

// ModelResponse represents a single model in the OpenAI /v1/models response.
type ModelResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
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
			result = append(result, ModelResponse{
				ID:      model.ID,
				Object:  "model",
				Created: now,
				OwnedBy: "proxy",
			})
		}
	}

	for _, endpoint := range p.endpoints {
		for _, model := range endpoint.Models {
			if !seen[model.ID] {
				seen[model.ID] = true
				result = append(result, ModelResponse{
					ID:      model.ID,
					Object:  "model",
					Created: now,
					OwnedBy: "proxy",
				})
			}
		}
	}

	return result
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
