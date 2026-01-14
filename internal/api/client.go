package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	apiURL = "https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels"
)

type QuotaInfo struct {
	RemainingFraction float64 `json:"remainingFraction"`
	ResetTime         string  `json:"resetTime"`
}

type ModelInfo struct {
	QuotaInfo QuotaInfo `json:"quotaInfo"`
}

type APIResponse struct {
	Models map[string]ModelInfo `json:"models"`
}

type ModelQuota struct {
	Name      string
	Quota     float64
	ResetTime time.Time
}

type AllModelsData struct {
	All    []ModelQuota
	Claude ModelQuota
	Gemini ModelQuota
}

func CallAPI(token string) (AllModelsData, error) {
	reqBody := bytes.NewBufferString("{}")

	req, err := http.NewRequest("POST", apiURL, reqBody)
	if err != nil {
		return AllModelsData{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "antigravity")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return AllModelsData{}, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return AllModelsData{}, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AllModelsData{}, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return AllModelsData{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	data := AllModelsData{}

	parseResetTime := func(t string) time.Time {
		if t == "" {
			return time.Time{}
		}
		parsed, err := time.Parse(time.RFC3339, t)
		if err != nil {
			return time.Time{}
		}
		return parsed
	}

	createModelQuota := func(name string, info ModelInfo) ModelQuota {
		return ModelQuota{
			Name:      name,
			Quota:     info.QuotaInfo.RemainingFraction,
			ResetTime: parseResetTime(info.QuotaInfo.ResetTime),
		}
	}

	for name, info := range apiResp.Models {
		data.All = append(data.All, createModelQuota(name, info))
	}

	sort.Slice(data.All, func(i, j int) bool {
		return data.All[i].Name < data.All[j].Name
	})

	if model, ok := apiResp.Models["claude-sonnet-4-5-thinking"]; ok {
		data.Claude = createModelQuota("claude-sonnet-4-5-thinking", model)
	}

	if model, ok := apiResp.Models["gemini-3-pro-high"]; ok {
		data.Gemini = createModelQuota("gemini-3-pro-high", model)
	}

	return data, nil
}

func IsClaudeModel(name string) bool {
	return strings.Contains(strings.ToLower(name), "claude")
}

func IsGeminiModel(name string) bool {
	return strings.Contains(strings.ToLower(name), "gemini")
}
