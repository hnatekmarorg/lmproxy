package proxy

import (
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
	return &Proxy{
		endpoints:          cfg.Endpoints,
		client:             &http.Client{Timeout: timeout},
		maxRequestBodySize: cfg.Server.MaxRequestBodySize,
		timeout:            timeout,
	}
}

func (p *Proxy) Handler(w http.ResponseWriter, r *http.Request) {
	_, modelConfig, targetURL := p.resolveTargetURL(r.URL)
	if targetURL == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	targetPath := p.computeTargetPath(r.URL.Path, modelConfig)
	targetURL.Path = targetPath
	targetURL.RawQuery = r.URL.RawQuery

	body, err := prepareRequestBody(r, modelConfig, targetPath, p.maxRequestBodySize)
	if err != nil {
		// Check for JSON parsing errors (syntax errors, not "invalid JSON" string)
		if strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "json:") {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	proxyReq, err := buildProxyRequest(r, targetURL, body)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	copyHeaders(proxyReq.Header, r.Header)

	resp, err := p.client.Do(proxyReq)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	forwardResponseBody(w, resp)
}
