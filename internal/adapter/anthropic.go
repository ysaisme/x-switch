package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type AnthropicAdapter struct{}

func (a *AnthropicAdapter) Name() string {
	return "anthropic"
}

func (a *AnthropicAdapter) ConvertRequest(baseURL string, apiKey string, req *UnifiedRequest) (*http.Request, error) {
	url := strings.TrimRight(baseURL, "/") + "/v1/messages"

	var openaiReq map[string]interface{}
	if err := json.Unmarshal(req.Body, &openaiReq); err != nil {
		return nil, fmt.Errorf("parse openai request: %w", err)
	}

	anthropicReq := map[string]interface{}{
		"model":      openaiReq["model"],
		"max_tokens": 4096,
	}

	if mt, ok := openaiReq["max_tokens"]; ok {
		anthropicReq["max_tokens"] = mt
	}

	if messages, ok := openaiReq["messages"]; ok {
		anthropicReq["messages"] = messages
	}

	if stream, ok := openaiReq["stream"]; ok {
		anthropicReq["stream"] = stream
	}

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("marshal anthropic request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	return httpReq, nil
}

func (a *AnthropicAdapter) ConvertResponse(resp *http.Response) (*UnifiedResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var anthropicResp map[string]interface{}
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	openaiResp := a.convertToOpenAIFormat(anthropicResp)

	resultBody, _ := json.Marshal(openaiResp)
	return &UnifiedResponse{Body: resultBody}, nil
}

func (a *AnthropicAdapter) convertToOpenAIFormat(anthropic map[string]interface{}) map[string]interface{} {
	model, _ := anthropic["model"].(string)
	id, _ := anthropic["id"].(string)

	choices := []interface{}{}
	if content, ok := anthropic["content"]; ok {
		if contentArr, ok := content.([]interface{}); ok {
			text := ""
			for _, item := range contentArr {
				if block, ok := item.(map[string]interface{}); ok {
					if block["type"] == "text" {
						if t, ok := block["text"].(string); ok {
							text += t
						}
					}
				}
			}
			choices = append(choices, map[string]interface{}{
				"index":         0,
				"finish_reason": "stop",
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": text,
				},
			})
		}
	}

	usage := map[string]interface{}{}
	if u, ok := anthropic["usage"]; ok {
		if usageMap, ok := u.(map[string]interface{}); ok {
			if it, ok := usageMap["input_tokens"]; ok {
				usage["prompt_tokens"] = it
			}
			if ot, ok := usageMap["output_tokens"]; ok {
				usage["completion_tokens"] = ot
			}
		}
	}

	return map[string]interface{}{
		"id":      id,
		"object":  "chat.completion",
		"model":   model,
		"choices": choices,
		"usage":   usage,
	}
}

func (a *AnthropicAdapter) ConvertStreamChunk(chunk []byte) ([]byte, error) {
	line := strings.TrimSpace(string(chunk))
	if !strings.HasPrefix(line, "data: ") {
		return chunk, nil
	}

	data := strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		return []byte("data: [DONE]\n\n"), nil
	}

	var event map[string]interface{}
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return chunk, nil
	}

	eventType, _ := event["type"].(string)

	switch eventType {
	case "content_block_delta":
		delta, _ := event["delta"].(map[string]interface{})
		text, _ := delta["text"].(string)
		openaiChunk := map[string]interface{}{
			"object": "chat.completion.chunk",
			"choices": []interface{}{
				map[string]interface{}{
					"index":         0,
					"finish_reason": nil,
					"delta": map[string]interface{}{
						"content": text,
					},
				},
			},
		}
		b, _ := json.Marshal(openaiChunk)
		return []byte("data: " + string(b) + "\n\n"), nil

	case "message_stop":
		doneChunk := map[string]interface{}{
			"object": "chat.completion.chunk",
			"choices": []interface{}{
				map[string]interface{}{
					"index":         0,
					"finish_reason": "stop",
					"delta":         map[string]interface{}{},
				},
			},
		}
		b, _ := json.Marshal(doneChunk)
		return []byte("data: " + string(b) + "\n\ndata: [DONE]\n\n"), nil

	default:
		return nil, nil
	}
}

func (a *AnthropicAdapter) ParseBalance(body []byte) (*BalanceInfo, error) {
	return &BalanceInfo{
		Currency: "USD",
		Raw:      string(body),
	}, nil
}
