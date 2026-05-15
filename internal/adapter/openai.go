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

func (a *OpenAIAdapter) TestConnectivity(baseURL string, apiKey string) (*ConnectivityResult, error) {
	url := strings.TrimRight(baseURL, "/") + "/v1/models"
	start := time.Now()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &ConnectivityResult{Ok: false, Error: err.Error()}, nil
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return &ConnectivityResult{Ok: false, LatencyMs: latency, Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return &ConnectivityResult{Ok: false, LatencyMs: latency, Error: "unauthorized: invalid API key"}, nil
	}
	if resp.StatusCode != 200 {
		return &ConnectivityResult{Ok: false, LatencyMs: latency, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}, nil
	}

	body, _ := io.ReadAll(resp.Body)
	var modelsResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(body, &modelsResp)

	return &ConnectivityResult{Ok: true, LatencyMs: latency, ModelsAvailable: len(modelsResp.Data)}, nil
}

func (a *OpenAIAdapter) ListModels(baseURL string, apiKey string) ([]ModelInfo, error) {
	url := strings.TrimRight(baseURL, "/") + "/v1/models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

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
		Data []struct {
			ID      string `json:"id"`
			Created int64  `json:"created"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	models := make([]ModelInfo, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, ModelInfo{ID: m.ID, Name: m.ID, Created: m.Created})
	}
	return models, nil
}
