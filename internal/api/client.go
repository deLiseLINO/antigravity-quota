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
	apiURL          = "https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels"
	quotaSummaryURL = "https://cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary"
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

// QuotaSummaryData is the parsed response of the retrieveUserQuotaSummary endpoint.
type QuotaSummaryData struct {
	Groups      []QuotaGroup
	Description string
}

// QuotaGroup is a family of quota buckets (e.g. "Gemini Models").
type QuotaGroup struct {
	DisplayName string
	Description string
	Buckets     []QuotaBucket
}

// QuotaBucket is a single quota limit within a group (e.g. weekly or 5h).
type QuotaBucket struct {
	BucketID          string
	DisplayName       string
	Description       string
	Window            string
	RemainingFraction float64
	RemainingAmount   float64
	ResetTime         time.Time
	Disabled          bool
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

	data.Claude = aggregateFamily(data.All, IsClaudeModel, "Claude")
	data.Gemini = aggregateFamily(data.All, IsGeminiModel, "Gemini")

	return data, nil
}

type quotaSummaryResponse struct {
	Groups []struct {
		DisplayName string `json:"displayName"`
		Description string `json:"description"`
		Buckets     []struct {
			BucketID          string  `json:"bucketId"`
			DisplayName       string  `json:"displayName"`
			Description       string  `json:"description"`
			Window            string  `json:"window"`
			RemainingFraction float64 `json:"remainingFraction"`
			RemainingAmount   float64 `json:"remainingAmount"`
			ResetTime         string  `json:"resetTime"`
			Disabled          bool    `json:"disabled"`
		} `json:"buckets"`
	} `json:"groups"`
	Description string `json:"description"`
}

// CallQuotaSummary fetches the user quota summary (groups with weekly/5h buckets)
// from the Antigravity API using the provided token.
func CallQuotaSummary(token string) (QuotaSummaryData, error) {
	reqBody := bytes.NewBufferString(`{"project": ""}`)

	req, err := http.NewRequest("POST", quotaSummaryURL, reqBody)
	if err != nil {
		return QuotaSummaryData{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "antigravity")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return QuotaSummaryData{}, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return QuotaSummaryData{}, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return QuotaSummaryData{}, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResp quotaSummaryResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return QuotaSummaryData{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

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

	data := QuotaSummaryData{Description: apiResp.Description}
	for _, g := range apiResp.Groups {
		group := QuotaGroup{
			DisplayName: g.DisplayName,
			Description: g.Description,
		}
		for _, b := range g.Buckets {
			group.Buckets = append(group.Buckets, QuotaBucket{
				BucketID:          b.BucketID,
				DisplayName:       b.DisplayName,
				Description:       b.Description,
				Window:            b.Window,
				RemainingFraction: b.RemainingFraction,
				RemainingAmount:   b.RemainingAmount,
				ResetTime:         parseResetTime(b.ResetTime),
				Disabled:          b.Disabled,
			})
		}
		data.Groups = append(data.Groups, group)
	}

	return data, nil
}

func aggregateFamily(models []ModelQuota, isMember func(string) bool, label string) ModelQuota {
	var members []ModelQuota
	for _, m := range models {
		if isMember(m.Name) {
			members = append(members, m)
		}
	}
	if len(members) == 0 {
		return ModelQuota{}
	}

	minQuota := members[0].Quota
	var earliestReset time.Time

	for _, m := range members {
		if m.Quota < minQuota {
			minQuota = m.Quota
		}
		if !m.ResetTime.IsZero() {
			if earliestReset.IsZero() || m.ResetTime.Before(earliestReset) {
				earliestReset = m.ResetTime
			}
		}
	}

	return ModelQuota{
		Name:      label,
		Quota:     minQuota,
		ResetTime: earliestReset,
	}
}

func IsClaudeModel(name string) bool {
	return strings.Contains(strings.ToLower(name), "claude")
}

func IsGeminiModel(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "gemini") && !strings.Contains(lower, "gemini-claude")
}
