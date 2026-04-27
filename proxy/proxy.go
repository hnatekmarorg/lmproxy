package proxy

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/hnatekmarorg/lmproxy/config"
)

type Proxy struct {
	endpoints          []config.Endpoint
	client             *http.Client
	maxRequestBodySize int
	timeout            time.Duration
}

func NewProxy(cfg *config.Config) *Proxy {
	timeout := time.Duration(cfg.Server.Timeout) * time.Second

	// Use Transport with separate timeouts for connection vs. body read
	// This allows long-running SSE streams while still timing out slow connections
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100, // Increase for many concurrent streams
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: timeout, // Timeout for getting response headers only
		ExpectContinueTimeout: 1 * time.Second,
		// Disable compression to pass through backend responses as-is
		DisableCompression: true,
	}

	// Create client with transport - no overall timeout to allow long streams
	// The ResponseHeaderTimeout in transport handles initial connection timeout
	client := &http.Client{
		Transport: transport,
	}

	return &Proxy{
		endpoints:          cfg.Endpoints,
		client:             client,
		maxRequestBodySize: cfg.Server.MaxRequestBodySize,
		timeout:            timeout,
	}
}

func (p *Proxy) Handler(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	startTime := time.Now()

	slog.Info("Request received", "request_id", requestID, "method", r.Method, "path", r.URL.Path, "remote_addr", r.RemoteAddr)

	_, modelConfig, targetURL := p.resolveTargetURL(r.URL, requestID)
	if targetURL == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		slog.Info("Request completed", "request_id", requestID, "status", http.StatusNotFound, "duration_ms", time.Since(startTime).Milliseconds(), "response_size", 0)
		return
	}

	targetPath := p.computeTargetPath(r.URL.Path, modelConfig)
	targetURL.Path = targetPath
	targetURL.RawQuery = r.URL.RawQuery

	slog.Debug("Forwarding request to backend", "request_id", requestID, "target_url", targetURL.String(), "method", r.Method)

	body, err := prepareRequestBody(r, modelConfig, targetPath, p.maxRequestBodySize, requestID)
	if err != nil {
		// Check for JSON parsing errors (syntax errors, not "invalid JSON" string)
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "json:") {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		slog.Info("Request completed", "request_id", requestID, "status", status, "duration_ms", time.Since(startTime).Milliseconds(), "response_size", 0, "error", err.Error())
		return
	}

	proxyReq, err := buildProxyRequest(r, targetURL, body)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		slog.Info("Request completed", "request_id", requestID, "status", http.StatusBadGateway, "duration_ms", time.Since(startTime).Milliseconds(), "response_size", 0, "error", err.Error())
		return
	}

	copyHeaders(proxyReq.Header, r.Header)

	resp, err := p.client.Do(proxyReq)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		slog.Info("Request completed", "request_id", requestID, "status", http.StatusBadGateway, "duration_ms", time.Since(startTime).Milliseconds(), "response_size", 0, "error", err.Error())
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	responseSize := forwardResponseBody(w, resp, requestID, startTime)
	slog.Info("Request completed", "request_id", requestID, "status", resp.StatusCode, "duration_ms", time.Since(startTime).Milliseconds(), "response_size", responseSize)
}
