package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type OpenAIAdapter struct{}

func (a *OpenAIAdapter) Name() string {
	return "openai"
}

func (a *OpenAIAdapter) ConvertRequest(baseURL string, apiKey string, req *UnifiedRequest) (*http.Request, error) {
	url := strings.TrimRight(baseURL, "/") + "/v1/chat/completions"

	body := req.Body
	if body == nil {
		payload := map[string]interface{}{
			"model":    req.Model,
			"messages": req.Messages,
		}
		if req.Stream {
			payload["stream"] = true
		}
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	return httpReq, nil
}

func (a *OpenAIAdapter) ConvertResponse(resp *http.Response) (*UnifiedResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result UnifiedResponse
	if err := json.Unmarshal(body, &result.Body); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

func (a *OpenAIAdapter) ConvertStreamChunk(chunk []byte) ([]byte, error) {
	return chunk, nil
}

func (a *OpenAIAdapter) ParseBalance(body []byte) (*BalanceInfo, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse balance: %w", err)
	}

	info := &BalanceInfo{
		Currency: "USD",
		Raw:      string(body),
	}

	if total, ok := result["total_available"]; ok {
		if v, ok := total.(float64); ok {
			info.Balance = v
		}
	}

	return info, nil
}
