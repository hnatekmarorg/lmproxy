package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ModelConfig represents a model configuration
type ModelConfig struct {
	ID                 string                 `yaml:"id"`
	Body               map[string]interface{} `yaml:"body"`
	Path               string                 `yaml:"path"`
	ExtraBody          map[string]interface{} `yaml:"extra_body"`
	ChatTemplateKwargs map[string]interface{} `yaml:"chat_template_kwargs"`
}

// Endpoint represents a proxy endpoint configuration
type Endpoint struct {
	Host   string        `yaml:"host"`
	Models []ModelConfig `yaml:"models"`
}

// Config holds the YAML configuration
type Config struct {
	Endpoints []Endpoint `yaml:"endpoints"`
}

var (
	endpointMap     = make(map[string]Endpoint)
	defaultEndpoint Endpoint
)

func loadConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	if len(config.Endpoints) == 0 {
		return fmt.Errorf("no endpoints configured")
	}

	defaultEndpoint = config.Endpoints[0]

	for _, endpoint := range config.Endpoints {
		endpointMap[endpoint.Host] = endpoint
	}

	fmt.Printf("Loaded %d endpoints from config\n", len(config.Endpoints))
	return nil
}

func findModelForPath(endpoint Endpoint, path string) *ModelConfig {
	for _, model := range endpoint.Models {
		if model.Path != "" && strings.HasPrefix(path, model.Path) {
			return &model
		}
	}
	return nil
}

func proxyHandler(clientRes http.ResponseWriter, clientReq *http.Request) {
	// Use the first endpoint (or default) since we're routing by path, not host
	endpoint := defaultEndpoint

	// Parse the target endpoint URL
	targetURL, err := url.Parse(endpoint.Host)
	if err != nil {
		http.Error(clientRes, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// Find matching model configuration based on URL prefix
	modelConfig := findModelForPath(endpoint, clientReq.URL.Path)

	// Determine target path by stripping the matched prefix (e.g., "/fast")
	targetPath := clientReq.URL.Path
	if modelConfig != nil && modelConfig.Path != "" {
		targetPath = strings.TrimPrefix(clientReq.URL.Path, modelConfig.Path)

		// Fallback: If they hit exactly `/fast` (like your old curl), default to chat completions
		if targetPath == "" || targetPath == "/" {
			targetPath = "/v1/chat/completions"
		} else if !strings.HasPrefix(targetPath, "/") {
			targetPath = "/" + targetPath
		}
	}

	targetURL.Path = targetPath
	targetURL.RawQuery = clientReq.URL.RawQuery

	var body io.Reader

	// Check if this is an inference endpoint where we should inject configuration
	isInferenceEndpoint := strings.HasSuffix(targetPath, "/completions") || strings.HasSuffix(targetPath, "/generate")

	// Only merge config body with request body for POST requests on inference endpoints
	if modelConfig != nil && clientReq.Method == "POST" && isInferenceEndpoint {
		var requestBody map[string]interface{}
		if clientReq.Body != nil {
			bodyBytes, err := io.ReadAll(clientReq.Body)
			if err == nil && len(bodyBytes) > 0 {
				json.Unmarshal(bodyBytes, &requestBody)
			}
		}

		if requestBody == nil {
			requestBody = make(map[string]interface{})
		}

		// Merge configured body parameters
		for k, v := range modelConfig.Body {
			requestBody[k] = v
		}

		// Handle extra_body for vLLM-specific parameters
		if len(modelConfig.ExtraBody) > 0 {
			extraBodyMap, ok := requestBody["extra_body"].(map[string]interface{})
			if !ok {
				extraBodyMap = make(map[string]interface{})
				requestBody["extra_body"] = extraBodyMap
			}
			for k, v := range modelConfig.ExtraBody {
				extraBodyMap[k] = v
			}
		}

		// Handle chat_template_kwargs at root level
		if len(modelConfig.ChatTemplateKwargs) > 0 {
			if _, exists := requestBody["chat_template_kwargs"]; !exists {
				requestBody["chat_template_kwargs"] = make(map[string]interface{})
			}
			if chatKwargs, ok := requestBody["chat_template_kwargs"].(map[string]interface{}); ok {
				for k, v := range modelConfig.ChatTemplateKwargs {
					chatKwargs[k] = v
				}
			}
		}

		bodyBytes, _ := json.Marshal(requestBody)
		body = bytes.NewBuffer(bodyBytes)
	} else {
		// Passthrough exactly as-is for GET requests (like /v1/models) or non-inference endpoints
		body = clientReq.Body
	}

	// Create the outbound request
	proxyReq, err := http.NewRequest(clientReq.Method, targetURL.String(), body)
	if err != nil {
		http.Error(clientRes, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// Copy headers from client request
	for key, values := range clientReq.Header {
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Set Host header to the target backend
	proxyReq.Host = targetURL.Host

	// Create HTTP client
	client := &http.Client{
		Timeout: 0, // No timeout for streaming
	}

	// Send the request
	proxyRes, err := client.Do(proxyReq)
	if err != nil {
		http.Error(clientRes, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer proxyRes.Body.Close()

	// Copy response headers
	for key, values := range proxyRes.Header {
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			clientRes.Header().Add(key, value)
		}
	}
	clientRes.WriteHeader(proxyRes.StatusCode)

	// Handle SSE streaming with keep-alives
	contentType := proxyRes.Header.Get("Content-Type")
	isSSE := strings.Contains(contentType, "text/event-stream")

	if isSSE {
		flusher, ok := clientRes.(http.Flusher)

		// Create a manual copy loop to flush after every read
		buf := make([]byte, 4096)
		for {
			n, err := proxyRes.Body.Read(buf)
			if n > 0 {
				clientRes.Write(buf[:n])
				if ok {
					flusher.Flush() // Push the chunk to the client immediately
				}
			}
			if err != nil {
				break // End of stream or error
			}
		}
	} else {
		io.Copy(clientRes, proxyRes.Body)
	}
}

// isHopByHopHeader returns true if the header is hop-by-hop and should not be forwarded
func isHopByHopHeader(header string) bool {
	hopByHopHeaders := map[string]bool{
		"Connection":          true,
		"Keep-Alive":          true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"Te":                  true,
		"Trailer":             true,
		"Transfer-Encoding":   true,
		"Upgrade":             true,
	}
	return hopByHopHeaders[header]
}

func main() {
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	if err := loadConfig(configPath); err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	http.HandleFunc("/", proxyHandler)

	fmt.Println("Proxy server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
