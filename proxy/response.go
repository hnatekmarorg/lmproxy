package proxy

import (
	"fmt"
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
		slog.Error("Response writer does not support flushing", "request_id", requestID)
		bytesWritten, _ := io.Copy(w, responseBody)
		return bytesWritten
	}

	buf := make([]byte, 64*1024) // 64KB buffer for headroom
	var bytesWritten int64
	var lastLogTime = startTime

	// Phantom chunk to keep AI SDK watchdog happy
	// Valid OpenAI chat.completion.chunk with empty delta - SDK parses it and resets timer
	const phantomChunk = `data: {"id":"keepalive","object":"chat.completion.chunk","created":0,"model":"keepalive","choices":[{"index":0,"delta":{}}]}\n\n`

	// Start keepalive timer - send phantom chunks every 5 seconds
	keepaliveTicker := time.NewTicker(5 * time.Second)
	defer keepaliveTicker.Stop()

	// Channel to signal completion
	done := make(chan struct{})
	defer close(done)

	// Goroutine to send phantom chunks
	go func() {
		for {
			select {
			case <-keepaliveTicker.C:
				// Send phantom chunk to reset AI SDK watchdog
				w.Write([]byte(phantomChunk))
				flusher.Flush()
			case <-done:
				return
			}
		}
	}()

	for {
		n, err := responseBody.Read(buf)
		if n > 0 {
			_, writeErr := w.Write(buf[:n])
			if writeErr != nil {
				slog.Error("Error writing to client during SSE stream", "request_id", requestID, "error", writeErr)
				break
			}
			flusher.Flush()
			bytesWritten += int64(n)

			// Log progress every 30 seconds for long streams
			if time.Since(lastLogTime) > 30*time.Second {
				slog.Debug("Stream in progress", "request_id", requestID, "bytes_written", bytesWritten, "duration_ms", time.Since(startTime).Milliseconds())
				lastLogTime = time.Now()
			}
		}
		if err != nil {
			if err == io.EOF {
				slog.Debug("Stream ended normally (EOF)", "request_id", requestID, "bytes_written", bytesWritten)
			} else {
				// Log the specific error to help diagnose timeouts
				slog.Error("Error reading from backend during SSE stream", "request_id", requestID, "error", err, "error_type", fmt.Sprintf("%T", err))
			}
			break
		}
	}
	slog.Info("Stream completed", "request_id", requestID, "bytes_written", bytesWritten, "duration_ms", time.Since(startTime).Milliseconds())
	return bytesWritten
}
