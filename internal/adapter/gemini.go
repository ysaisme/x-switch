package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type GeminiAdapter struct{}

func (a *GeminiAdapter) Name() string {
	return "gemini"
}

func (a *GeminiAdapter) ConvertRequest(baseURL string, apiKey string, req *UnifiedRequest) (*http.Request, error) {
	url := strings.TrimRight(baseURL, "/") + "/v1beta/models/" + req.Model + ":generateContent?key=" + apiKey

	var openaiReq map[string]interface{}
	if err := json.Unmarshal(req.Body, &openaiReq); err != nil {
		return nil, fmt.Errorf("parse openai request: %w", err)
	}

	geminiReq := map[string]interface{}{}

	if messages, ok := openaiReq["messages"]; ok {
		contents := a.convertMessagesToGemini(messages)
		geminiReq["contents"] = contents
	}

	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal gemini request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	return httpReq, nil
}

func (a *GeminiAdapter) convertMessagesToGemini(messages interface{}) []interface{} {
	msgArr, ok := messages.([]interface{})
	if !ok {
		return nil
	}

	var contents []interface{}
	for _, msg := range msgArr {
		m, ok := msg.(map[string]interface{})
		if !ok {
			continue
		}

		role, _ := m["role"].(string)
		content, _ := m["content"].(string)

		geminiRole := "user"
		if role == "assistant" {
			geminiRole = "model"
		}

		contents = append(contents, map[string]interface{}{
			"role": geminiRole,
			"parts": []interface{}{
				map[string]interface{}{
					"text": content,
				},
			},
		})
	}

	return contents
}

func (a *GeminiAdapter) ConvertResponse(resp *http.Response) (*UnifiedResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp map[string]interface{}
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	openaiResp := a.convertToOpenAIFormat(geminiResp)

	resultBody, _ := json.Marshal(openaiResp)
	return &UnifiedResponse{Body: resultBody}, nil
}

func (a *GeminiAdapter) convertToOpenAIFormat(gemini map[string]interface{}) map[string]interface{} {
	text := ""
	if candidates, ok := gemini["candidates"].([]interface{}); ok && len(candidates) > 0 {
		if candidate, ok := candidates[0].(map[string]interface{}); ok {
			if content, ok := candidate["content"].(map[string]interface{}); ok {
				if parts, ok := content["parts"].([]interface{}); ok {
					for _, part := range parts {
						if p, ok := part.(map[string]interface{}); ok {
							if t, ok := p["text"].(string); ok {
								text += t
							}
						}
					}
				}
			}
		}
	}

	usage := map[string]interface{}{}
	if u, ok := gemini["usageMetadata"]; ok {
		if usageMap, ok := u.(map[string]interface{}); ok {
			if pt, ok := usageMap["promptTokenCount"]; ok {
				usage["prompt_tokens"] = pt
			}
			if ct, ok := usageMap["candidatesTokenCount"]; ok {
				usage["completion_tokens"] = ct
			}
		}
	}

	return map[string]interface{}{
		"id":     "gemini-" + fmt.Sprintf("%d", len(text)),
		"object": "chat.completion",
		"choices": []interface{}{
			map[string]interface{}{
				"index":         0,
				"finish_reason": "stop",
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": text,
				},
			},
		},
		"usage": usage,
	}
}

func (a *GeminiAdapter) ConvertStreamChunk(chunk []byte) ([]byte, error) {
	line := strings.TrimSpace(string(chunk))
	if !strings.HasPrefix(line, "data: ") {
		return chunk, nil
	}

	data := strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		return []byte("data: [DONE]\n\n"), nil
	}

	var geminiChunk map[string]interface{}
	if err := json.Unmarshal([]byte(data), &geminiChunk); err != nil {
		return chunk, nil
	}

	text := ""
	if candidates, ok := geminiChunk["candidates"].([]interface{}); ok && len(candidates) > 0 {
		if candidate, ok := candidates[0].(map[string]interface{}); ok {
			if content, ok := candidate["content"].(map[string]interface{}); ok {
				if parts, ok := content["parts"].([]interface{}); ok {
					for _, part := range parts {
						if p, ok := part.(map[string]interface{}); ok {
							if t, ok := p["text"].(string); ok {
								text += t
							}
						}
					}
				}
			}
		}
	}

	if text == "" {
		return nil, nil
	}

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
}

func (a *GeminiAdapter) TestConnectivity(baseURL string, apiKey string) (*ConnectivityResult, error) {
	url := strings.TrimRight(baseURL, "/") + "/v1beta/models?key=" + apiKey
	start := time.Now()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &ConnectivityResult{Ok: false, Error: err.Error()}, nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return &ConnectivityResult{Ok: false, LatencyMs: latency, Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 400 || resp.StatusCode == 403 {
		return &ConnectivityResult{Ok: false, LatencyMs: latency, Error: "unauthorized: invalid API key"}, nil
	}
	if resp.StatusCode != 200 {
		return &ConnectivityResult{Ok: false, LatencyMs: latency, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}, nil
	}

	body, _ := io.ReadAll(resp.Body)
	var modelsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	json.Unmarshal(body, &modelsResp)

	return &ConnectivityResult{Ok: true, LatencyMs: latency, ModelsAvailable: len(modelsResp.Models)}, nil
}

func (a *GeminiAdapter) ListModels(baseURL string, apiKey string) ([]ModelInfo, error) {
	url := strings.TrimRight(baseURL, "/") + "/v1beta/models?key=" + apiKey

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Models []struct {
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	models := make([]ModelInfo, 0, len(result.Models))
	for _, m := range result.Models {
		id := m.Name
		if strings.HasPrefix(id, "models/") {
			id = strings.TrimPrefix(id, "models/")
		}
		name := m.DisplayName
		if name == "" {
			name = id
		}
		models = append(models, ModelInfo{ID: id, Name: name})
	}
	return models, nil
}
