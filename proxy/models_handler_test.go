package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hnatekmarorg/lmproxy/config"
)

func TestHandleListModels_Default(t *testing.T) {
	// Test default behavior (reachableOnly=false) - returns all models
	p := &Proxy{
		endpoints: []config.Endpoint{
			{
				Host: "https://example.com",
				Models: []config.ModelConfig{
					{ID: "model-a", Body: map[string]interface{}{"model": "upstream-a"}},
					{ID: "model-b", Body: map[string]interface{}{"model": "upstream-b"}},
				},
			},
		},
		reachableOnly: false,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()
	p.handleListModels(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var listResp ListModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()

	if len(listResp.Data) != 2 {
		t.Fatalf("Expected 2 models, got %d", len(listResp.Data))
	}
	if listResp.Data[0].ID != "model-a" || listResp.Data[1].ID != "model-b" {
		t.Errorf("Unexpected model order: %+v", listResp.Data)
	}
}

func TestHandleListModels_ReachableOnly_NoUpstream(t *testing.T) {
	// Test reachableOnly=true with no reachable upstreams - returns empty
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(upstreamListModelsResponse{
			Object: "list",
			Data:   []upstreamModel{},
		})
	}))
	defer ts.Close()

	p := &Proxy{
		endpoints: []config.Endpoint{
			{
				Host: ts.URL,
				Models: []config.ModelConfig{
					{ID: "model-a", Body: map[string]interface{}{"model": "upstream-a"}},
					{ID: "model-b", Body: map[string]interface{}{"model": "upstream-b"}},
				},
			},
		},
		reachableOnly: true,
		client:        http.DefaultClient,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()
	p.handleListModels(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var listResp ListModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()

	if len(listResp.Data) != 0 {
		t.Fatalf("Expected 0 models (empty upstream), got %d", len(listResp.Data))
	}
}

func TestHandleListModels_ReachableOnly_Filtered(t *testing.T) {
	// Test reachableOnly=true - only returns models whose body.model matches upstream
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(upstreamListModelsResponse{
			Object: "list",
			Data:   []upstreamModel{{ID: "upstream-a"}},
		})
	}))
	defer ts.Close()

	p := &Proxy{
		endpoints: []config.Endpoint{
			{
				Host: ts.URL,
				Models: []config.ModelConfig{
					{ID: "model-a", Body: map[string]interface{}{"model": "upstream-a"}},
					{ID: "model-b", Body: map[string]interface{}{"model": "upstream-b"}},
				},
			},
		},
		reachableOnly: true,
		client:        http.DefaultClient,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()
	p.handleListModels(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var listResp ListModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()

	if len(listResp.Data) != 1 {
		t.Fatalf("Expected 1 model (only model-a reachable), got %d", len(listResp.Data))
	}
	if listResp.Data[0].ID != "model-a" {
		t.Errorf("Expected model-a, got %s", listResp.Data[0].ID)
	}
}

func TestHandleListModels_ReachableOnly_ModelIDFallback(t *testing.T) {
	// Test reachableOnly=true with body.model not set - falls back to model ID
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(upstreamListModelsResponse{
			Object: "list",
			Data:   []upstreamModel{{ID: "model-a"}},
		})
	}))
	defer ts.Close()

	p := &Proxy{
		endpoints: []config.Endpoint{
			{
				Host: ts.URL,
				Models: []config.ModelConfig{
					{ID: "model-a"}, // No body.model set
					{ID: "model-b"},
				},
			},
		},
		reachableOnly: true,
		client:        http.DefaultClient,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()
	p.handleListModels(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var listResp ListModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()

	if len(listResp.Data) != 1 {
		t.Fatalf("Expected 1 model (only model-a reachable by ID), got %d", len(listResp.Data))
	}
	if listResp.Data[0].ID != "model-a" {
		t.Errorf("Expected model-a, got %s", listResp.Data[0].ID)
	}
}

func TestHandleListModels_ReachableOnly_UpstreamUnreachable(t *testing.T) {
	// Test reachableOnly=true with upstream unreachable - returns empty
	p := &Proxy{
		endpoints: []config.Endpoint{
			{
				Host: "http://127.0.0.1:1", // Unreachable
				Models: []config.ModelConfig{
					{ID: "model-a", Body: map[string]interface{}{"model": "upstream-a"}},
				},
			},
		},
		reachableOnly: true,
		client:        http.DefaultClient,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()
	p.handleListModels(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var listResp ListModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()

	if len(listResp.Data) != 0 {
		t.Fatalf("Expected 0 models (upstream unreachable), got %d", len(listResp.Data))
	}
}

func TestHandleListModels_ReachableOnly_MultipleEndpoints(t *testing.T) {
	// Test reachableOnly=true with multiple endpoints
	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(upstreamListModelsResponse{
			Object: "list",
			Data:   []upstreamModel{{ID: "upstream-a"}},
		})
	}))
	defer ts1.Close()

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(upstreamListModelsResponse{
			Object: "list",
			Data:   []upstreamModel{{ID: "upstream-c"}},
		})
	}))
	defer ts2.Close()

	p := &Proxy{
		endpoints: []config.Endpoint{
			{
				Host: ts1.URL,
				Models: []config.ModelConfig{
					{ID: "model-a", Body: map[string]interface{}{"model": "upstream-a"}},
					{ID: "model-b", Body: map[string]interface{}{"model": "upstream-b"}}, // Not reachable
				},
			},
			{
				Host: ts2.URL,
				Models: []config.ModelConfig{
					{ID: "model-c", Body: map[string]interface{}{"model": "upstream-c"}},
				},
			},
		},
		reachableOnly: true,
		client:        http.DefaultClient,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()
	p.handleListModels(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var listResp ListModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	resp.Body.Close()

	if len(listResp.Data) != 2 {
		t.Fatalf("Expected 2 models (model-a and model-c), got %d: %+v", len(listResp.Data), listResp.Data)
	}
}

func TestHandleListModels_NotGET(t *testing.T) {
	p := &Proxy{}
	req := httptest.NewRequest(http.MethodPost, "/v1/models", nil)
	w := httptest.NewRecorder()
	p.handleListModels(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestGetAllModels_Deduplication(t *testing.T) {
	p := &Proxy{
		topLevelModels: []config.ModelConfig{
			{ID: "model-a"},
		},
		endpoints: []config.Endpoint{
			{
				Host: "https://example.com",
				Models: []config.ModelConfig{
					{ID: "model-a"}, // Duplicate - should be excluded
					{ID: "model-b"},
				},
			},
		},
	}

	models := p.getAllModels()
	if len(models) != 2 {
		t.Fatalf("Expected 2 models (deduplicated), got %d", len(models))
	}
}

func TestIsModelReachable_BodyModel(t *testing.T) {
	p := &Proxy{}
	reachableSet := map[string]bool{"upstream-a": true}

	// body.model matches
	model := config.ModelConfig{
		ID:   "model-a",
		Body: map[string]interface{}{"model": "upstream-a"},
	}
	if !p.isModelReachable(model, reachableSet) {
		t.Error("Expected model to be reachable via body.model")
	}

	// body.model doesn't match
	model2 := config.ModelConfig{
		ID:   "model-b",
		Body: map[string]interface{}{"model": "upstream-b"},
	}
	if p.isModelReachable(model2, reachableSet) {
		t.Error("Expected model to not be reachable")
	}
}

func TestIsModelReachable_IDFallback(t *testing.T) {
	p := &Proxy{}
	reachableSet := map[string]bool{"model-a": true}

	// No body.model, ID matches
	model := config.ModelConfig{ID: "model-a"}
	if !p.isModelReachable(model, reachableSet) {
		t.Error("Expected model to be reachable via ID fallback")
	}

	// No body.model, ID doesn't match
	model2 := config.ModelConfig{ID: "model-b"}
	if p.isModelReachable(model2, reachableSet) {
		t.Error("Expected model to not be reachable")
	}
}

func TestQueryAllUpstreams(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(upstreamListModelsResponse{
			Object: "list",
			Data:   []upstreamModel{{ID: "upstream-a"}, {ID: "upstream-b"}},
		})
	}))
	defer ts.Close()

	p := &Proxy{
		endpoints: []config.Endpoint{
			{Host: ts.URL, Models: []config.ModelConfig{{ID: "m1"}}},
			{Host: ts.URL, Models: []config.ModelConfig{{ID: "m2"}}},
		},
		client: http.DefaultClient,
	}

	result := p.queryAllUpstreams()
	if len(result) != 1 {
		t.Fatalf("Expected 1 upstream host in result, got %d", len(result))
	}

	models, ok := result[ts.URL]
	if !ok {
		t.Fatalf("Expected host %s in result", ts.URL)
	}
	if len(models) != 2 {
		t.Fatalf("Expected 2 model IDs from upstream, got %d", len(models))
	}
}
