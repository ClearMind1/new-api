package middleware

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// responseCaptureWriter wraps gin.ResponseWriter to capture the response body
type responseCaptureWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseCaptureWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseCaptureWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// Flush implements http.Flusher for streaming support
func (w *responseCaptureWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack implements http.Hijacker for WebSocket support
func (w *responseCaptureWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Push implements http.Pusher for HTTP/2 server push
func (w *responseCaptureWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

// parseSSEStream parses raw SSE stream data into a readable format.
// Returns both the combined text content and the raw data.
func parseSSEStream(raw string) string {
	var contentBuilder strings.Builder
	var reasoningBuilder strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			continue
		}

		var chunk map[string]interface{}
		if err := common.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		eventType, _ := chunk["type"].(string)

		// OpenAI format: choices[].delta.{content,reasoning_content}
		if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if delta, ok := choice["delta"].(map[string]interface{}); ok {
					if content, ok := delta["content"].(string); ok && content != "" {
						contentBuilder.WriteString(content)
					}
					if reasoning, ok := delta["reasoning_content"].(string); ok && reasoning != "" {
						reasoningBuilder.WriteString(reasoning)
					}
				}
			}
			continue
		}

		// Claude format: type=content_block_delta with delta.text or delta.thinking
		if eventType == "content_block_delta" {
			if delta, ok := chunk["delta"].(map[string]interface{}); ok {
				deltaType, _ := delta["type"].(string)
				switch deltaType {
				case "text_delta":
					if text, ok := delta["text"].(string); ok && text != "" {
						contentBuilder.WriteString(text)
					}
				case "thinking_delta":
					if thinking, ok := delta["thinking"].(string); ok && thinking != "" {
						reasoningBuilder.WriteString(thinking)
					}
				}
			}
			continue
		}

		// Claude format: type=content_block_start with content_block.text
		if eventType == "content_block_start" {
			if block, ok := chunk["content_block"].(map[string]interface{}); ok {
				if blockType, _ := block["type"].(string); blockType == "text" {
					if text, ok := block["text"].(string); ok && text != "" {
						contentBuilder.WriteString(text)
					}
				}
			}
			continue
		}
	}

	if contentBuilder.Len() == 0 && reasoningBuilder.Len() == 0 {
		return ""
	}

	var result strings.Builder
	if reasoningBuilder.Len() > 0 {
		result.WriteString("=== Reasoning ===\n")
		result.WriteString(reasoningBuilder.String())
		result.WriteString("\n\n=== Response ===\n")
	}
	result.WriteString(contentBuilder.String())

	return result.String()
}

// truncateEmbeddingResponse truncates large embedding vectors in non-streaming responses.
// Keeps the first 10 values of each embedding array and adds a note.
func truncateEmbeddingResponse(body string) string {
	// Quick check: if no "embedding" key, return as-is
	if !strings.Contains(body, "embedding") {
		return body
	}

	// If response is small enough, keep it
	if len(body) <= 8192 {
		return body
	}

	// Try to parse and truncate
	var resp map[string]interface{}
	if err := common.Unmarshal([]byte(body), &resp); err != nil {
		return body
	}

	// Check if this is an embedding response
	if data, ok := resp["data"].([]interface{}); ok && len(data) > 0 {
		if embedding, ok := data[0].(map[string]interface{})["embedding"].([]interface{}); ok && len(embedding) > 10 {
			// Truncate to first 10 values
			truncated := embedding[:10]
			data[0].(map[string]interface{})["embedding"] = truncated
			resp["data"] = data
			// Re-marshal
			result, err := common.Marshal(resp)
			if err == nil {
				return string(result) + "\n\n[embedding truncated: kept first 10 of " + fmt.Sprintf("%d", len(embedding)) + " values]"
			}
		}
	}

	// Fallback: just truncate the whole body
	return body
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ResponseCapture captures request and response details in one place to avoid race conditions.
// Captures: request body, request headers, response body (with SSE stream parsing).
// Only enabled when LOG_REQUEST_DETAIL=1.
func ResponseCapture() gin.HandlerFunc {
	return func(c *gin.Context) {
		if common.GetEnvOrDefault("LOG_REQUEST_DETAIL", 1) != 1 {
			c.Next()
			return
		}

		requestId := c.GetString(common.RequestIdKey)
		if requestId == "" {
			c.Next()
			return
		}

		// Extract request body BEFORE c.Next() (before BodyStorage is cleaned up)
		var requestBody string
		if storage, err := common.GetBodyStorage(c); err == nil {
			if bodyBytes, err := storage.Bytes(); err == nil {
				requestBody = string(bodyBytes)
				storage.Seek(0, 0)
			}
		}
		if len(requestBody) > 1024*1024 {
			requestBody = requestBody[:1024*1024] + "\n... [truncated]"
		}

		// Extract request headers (filter sensitive ones)
		skipHeaders := map[string]bool{
			"authorization": true,
			"x-api-key":     true,
			"cookie":        true,
		}
		headers := make(map[string]interface{})
		for k, v := range c.Request.Header {
			if skipHeaders[strings.ToLower(k)] {
				continue
			}
			headers[k] = strings.Join(v, ", ")
		}
		headersStr := common.MapToJsonStr(headers)
		requestPath := c.Request.URL.Path
		requestMethod := c.Request.Method
		modelName := c.GetString("original_model")
		userId := c.GetInt("id")

		// Wrap response writer
		captureWriter := &responseCaptureWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = captureWriter

		c.Next()

		// Extract response body AFTER c.Next()
		responseBody := captureWriter.body.String()
		if responseBody == "" && requestBody == "" {
			return
		}

		// Truncate if too large (1MB limit)
		if len(responseBody) > 1024*1024 {
			responseBody = responseBody[:1024*1024] + "\n... [truncated]"
		}

		// Parse SSE stream for streaming responses
		var processedResponse string
		isStream := strings.Contains(captureWriter.Header().Get("Content-Type"), "text/event-stream") ||
			strings.Contains(responseBody, "data: ")
		if isStream && responseBody != "" {
			processedResponse = parseSSEStream(responseBody)
		} else if responseBody != "" {
			// For non-streaming responses, check if it's an embedding response
			// and truncate large embedding vectors to save space
			processedResponse = truncateEmbeddingResponse(responseBody)
		}

		statusCode := captureWriter.Status()
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		// Save everything in ONE call to avoid race conditions
		detail := &model.LogDetail{
			RequestId:      requestId,
			RequestBody:    requestBody,
			RequestPath:    requestPath,
			RequestMethod:  requestMethod,
			RequestHeaders: headersStr,
			ResponseBody: func() string {
				if processedResponse != "" {
					return processedResponse
				}
				return responseBody
			}(),
			StatusCode: statusCode,
			ModelName:  modelName,
			UserId:     userId,
			CreatedAt:  common.GetTimestamp(),
		}

		if err := model.UpsertLogDetail(detail); err != nil {
			common.SysError("failed to save request detail: " + err.Error())
		}
	}
}
