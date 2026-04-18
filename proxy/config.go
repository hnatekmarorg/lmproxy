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

type Proxy struct {
	endpoints []config.Endpoint
	client    *http.Client
}

func NewProxy(cfg *config.Config) *Proxy {
	return &Proxy{
		endpoints: cfg.Endpoints,
		client:    &http.Client{Timeout: 0},
	}
}

func (p *Proxy) Handler(clientRes http.ResponseWriter, clientReq *http.Request) {
	// 1. Resolve target URL from config
	endpoint, targetURL := p.resolveTargetURL(clientReq.URL)
	if targetURL == nil {
		http.Error(clientRes, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// 2. Find matching model configuration
	modelConfig := p.findModelForPath(endpoint, clientReq.URL.Path)

	// 3. Compute target path (strip model prefix if present)
	targetPath := p.computeTargetPath(clientReq.URL.Path, modelConfig)
	targetURL.Path = targetPath
	targetURL.RawQuery = clientReq.URL.RawQuery

	// 4. Prepare request body (merge config if needed)
	body, err := prepareRequestBody(clientReq, modelConfig, targetPath)
	if err != nil {
		if strings.Contains(err.Error(), "invalid JSON") {
			http.Error(clientRes, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(clientRes, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// 5. Build proxy request
	proxyReq, err := buildProxyRequest(clientReq, targetURL, body)
	if err != nil {
		http.Error(clientRes, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// 6. Copy headers from client request
	copyHeaders(proxyReq.Header, clientReq.Header)

	// 7. Send request to backend
	proxyRes, err := p.client.Do(proxyReq)
	if err != nil {
		http.Error(clientRes, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer proxyRes.Body.Close()

	// 8. Copy response headers
	copyHeaders(clientRes.Header(), proxyRes.Header)
	clientRes.WriteHeader(proxyRes.StatusCode)

	// 9. Forward response body (handle SSE streaming)
	forwardResponseBody(clientRes, proxyRes)
}

func (p *Proxy) resolveTargetURL(requestURL *url.URL) (*config.Endpoint, *url.URL) {
	// Use the first endpoint as default
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

		// Default to chat completions if path is empty or root
		if targetPath == "" || targetPath == "/" {
			targetPath = "/v1/chat/completions"
		} else if !strings.HasPrefix(targetPath, "/") {
			targetPath = "/" + targetPath
		}
	}

	return targetPath
}

func prepareRequestBody(clientReq *http.Request, modelConfig *config.ModelConfig, targetPath string) (io.Reader, error) {
	// Check if this is an inference endpoint
	isInferenceEndpoint := strings.HasSuffix(targetPath, "/completions") || strings.HasSuffix(targetPath, "/generate")

	// Only merge config for POST requests on inference endpoints
	if modelConfig == nil || clientReq.Method != "POST" || !isInferenceEndpoint {
		return clientReq.Body, nil
	}

	// Skip if no merging is needed
	if len(modelConfig.Body) == 0 && len(modelConfig.ExtraBody) == 0 && len(modelConfig.ChatTemplateKwargs) == 0 {
		return clientReq.Body, nil
	}

	// Read and parse request body
	var requestBody map[string]interface{}
	if clientReq.Body != nil {
		bodyBytes, err := io.ReadAll(clientReq.Body)
		if err != nil {
			slog.Error("Failed to read request body", "error", err)
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
				slog.Error("Failed to parse request body as JSON", "error", err)
				return nil, fmt.Errorf("invalid JSON in request body: %w", err)
			}
		}
	}

	if requestBody == nil {
		requestBody = make(map[string]interface{})
	}

	// Merge configured body parameters
	util.MergeMap(requestBody, modelConfig.Body, "")
	util.MergeMap(requestBody, modelConfig.ExtraBody, "extra_body")
	util.MergeMap(requestBody, modelConfig.ChatTemplateKwargs, "chat_template_kwargs")

	// Marshal merged body
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		slog.Error("Failed to marshal request body", "error", err)
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	return bytes.NewBuffer(bodyBytes), nil
}

func buildProxyRequest(clientReq *http.Request, targetURL *url.URL, body io.Reader) (*http.Request, error) {
	proxyReq, err := http.NewRequest(clientReq.Method, targetURL.String(), body)
	if err != nil {
		return nil, err
	}

	proxyReq.Host = targetURL.Host
	return proxyReq, nil
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

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
	_, ok := hopByHopHeaders[header]
	return ok
}

func forwardResponseBody(clientRes http.ResponseWriter, proxyRes *http.Response) {
	isSSE := strings.Contains(proxyRes.Header.Get("Content-Type"), "text/event-stream")

	if isSSE {
		streamSSE(clientRes, proxyRes.Body)
	} else {
		io.Copy(clientRes, proxyRes.Body)
	}
}

func streamSSE(clientRes http.ResponseWriter, responseBody io.ReadCloser) {
	flusher, ok := clientRes.(http.Flusher)
	if !ok {
		io.Copy(clientRes, responseBody)
		return
	}

	buf := make([]byte, 4096)
	for {
		n, err := responseBody.Read(buf)
		if n > 0 {
			clientRes.Write(buf[:n])
			flusher.Flush()
		}
		if err != nil {
			if err != io.EOF {
				slog.Error("Error reading from backend during SSE stream", "error", err)
			}
			break
		}
	}
}
