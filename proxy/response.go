package proxy

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func forwardResponseBody(w http.ResponseWriter, resp *http.Response, requestID string, startTime time.Time) int64 {
	isSSE := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")

	slog.Info("Response started", "request_id", requestID, "status", resp.StatusCode)

	var responseSize int64
	if isSSE {
		responseSize = streamSSE(w, resp.Body, requestID, startTime)
	} else {
		var err error
		responseSize, err = io.Copy(w, resp.Body)
		if err != nil {
			slog.Error("Error copying response body", "request_id", requestID, "error", err)
		}
	}
	return responseSize
}

func streamSSE(w http.ResponseWriter, responseBody io.ReadCloser, requestID string, startTime time.Time) int64 {
	flusher, ok := w.(http.Flusher)
	if !ok {
		bytesWritten, _ := io.Copy(w, responseBody)
		return bytesWritten
	}

	buf := make([]byte, 64*1024) // 64KB buffer for headroom
	var bytesWritten int64
	for {
		n, err := responseBody.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
			flusher.Flush()
			bytesWritten += int64(n)
		}
		if err != nil {
			if err != io.EOF {
				slog.Error("Error reading from backend during SSE stream", "request_id", requestID, "error", err)
			}
			break
		}
	}
	slog.Info("Stream completed", "request_id", requestID, "bytes_written", bytesWritten, "duration_ms", time.Since(startTime).Milliseconds())
	return bytesWritten
}
