package proxy

import (
	"net/http"
	"strings"

	"github.com/hnatekmarorg/lmproxy/config"
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

func (p *Proxy) Handler(w http.ResponseWriter, r *http.Request) {
	_, modelConfig, targetURL := p.resolveTargetURL(r.URL)
	if targetURL == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	targetPath := p.computeTargetPath(r.URL.Path, modelConfig)
	targetURL.Path = targetPath
	targetURL.RawQuery = r.URL.RawQuery

	body, err := prepareRequestBody(r, modelConfig, targetPath)
	if err != nil {
		if strings.Contains(err.Error(), "invalid JSON") {
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
