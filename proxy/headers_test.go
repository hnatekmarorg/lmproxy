package proxy

import (
	"net/http"
	"testing"
)

func TestIsHopByHopHeader(t *testing.T) {
	hopByHop := []string{"Connection", "Keep-Alive", "Transfer-Encoding", "Upgrade"}
	nonHop := []string{"Content-Type", "Authorization", "User-Agent"}

	for _, h := range hopByHop {
		if !isHopByHopHeader(h) {
			t.Errorf("Expected %q to be hop-by-hop", h)
		}
	}

	for _, h := range nonHop {
		if isHopByHopHeader(h) {
			t.Errorf("Expected %q to NOT be hop-by-hop", h)
		}
	}
}

func TestCopyHeaders(t *testing.T) {
	src := http.Header{
		"Content-Type":    []string{"application/json"},
		"Authorization":   []string{"Bearer token"},
		"Connection":      []string{"keep-alive"},
		"X-Custom-Header": []string{"value"},
	}

	dst := make(http.Header)
	copyHeaders(dst, src)

	if dst.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type to be copied")
	}
	if dst.Get("Authorization") != "Bearer token" {
		t.Errorf("Expected Authorization to be copied")
	}
	if dst.Get("Connection") != "" {
		t.Errorf("Expected Connection to be skipped (hop-by-hop)")
	}
	if dst.Get("X-Custom-Header") != "value" {
		t.Errorf("Expected X-Custom-Header to be copied")
	}
}
