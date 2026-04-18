package proxy

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
)

func forwardResponseBody(w http.ResponseWriter, resp *http.Response) {
	isSSE := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")

	if isSSE {
		streamSSE(w, resp.Body)
	} else {
		io.Copy(w, resp.Body)
	}
}

func streamSSE(w http.ResponseWriter, responseBody io.ReadCloser) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		io.Copy(w, responseBody)
		return
	}

	buf := make([]byte, 4096)
	for {
		n, err := responseBody.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
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
