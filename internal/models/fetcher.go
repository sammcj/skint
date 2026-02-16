package models

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// ModelInfo represents a model available from a provider.
type ModelInfo struct {
	ID          string
	DisplayName string // optional, falls back to ID
	Created     int64  // unix timestamp, 0 if unknown
}

// Label returns the display name if set, otherwise the ID.
func (m ModelInfo) Label() string {
	if m.DisplayName != "" {
		return m.DisplayName
	}
	return m.ID
}

// FetchResult holds the result of a model fetch operation.
type FetchResult struct {
	Models []ModelInfo
	Err    error
}

// fetchTimeout is the HTTP client timeout for model fetches.
const fetchTimeout = 5 * time.Second

// FetchModels fetches available models from a provider endpoint.
// The strategy is determined by provider name and type.
func FetchModels(baseURL, apiKey, providerName string) FetchResult {
	strategy := selectStrategy(baseURL, providerName)
	if strategy == nil {
		return FetchResult{}
	}
	return strategy(baseURL, apiKey)
}

type fetchFunc func(baseURL, apiKey string) FetchResult

func selectStrategy(baseURL, providerName string) fetchFunc {
	switch providerName {
	case "native", "anthropic":
		// Anthropic models are well known; no listing endpoint needed.
		return nil
	case "ollama":
		return fetchOllama
	case "openrouter":
		return fetchOpenRouter
	case "llamacpp":
		// llama.cpp may or may not support /v1/models; try it but tolerate failure.
		return fetchOpenAICompatibleSilent
	default:
		if baseURL == "" {
			return nil
		}
		return fetchOpenAICompatible
	}
}

// fetchOpenAICompatible fetches models from an OpenAI-compatible /v1/models endpoint.
func fetchOpenAICompatible(baseURL, apiKey string) FetchResult {
	trimmed := strings.TrimRight(baseURL, "/")
	var url string
	if strings.HasSuffix(trimmed, "/v1") {
		url = trimmed + "/models"
	} else {
		url = trimmed + "/v1/models"
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return FetchResult{Err: fmt.Errorf("creating request: %w", err)}
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	return doOpenAIModelsRequest(req)
}

// fetchOpenAICompatibleSilent is like fetchOpenAICompatible but returns empty on error
// instead of propagating the error (for providers that may not support the endpoint).
func fetchOpenAICompatibleSilent(baseURL, apiKey string) FetchResult {
	result := fetchOpenAICompatible(baseURL, apiKey)
	if result.Err != nil {
		return FetchResult{}
	}
	return result
}

func doOpenAIModelsRequest(req *http.Request) FetchResult {
	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return FetchResult{Err: fmt.Errorf("fetching models: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return FetchResult{Err: fmt.Errorf("models endpoint returned status %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return FetchResult{Err: fmt.Errorf("reading response: %w", err)}
	}

	var response struct {
		Data []struct {
			ID      string `json:"id"`
			Created int64  `json:"created"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return FetchResult{Err: fmt.Errorf("parsing response: %w", err)}
	}

	models := make([]ModelInfo, 0, len(response.Data))
	for _, m := range response.Data {
		if m.ID != "" {
			models = append(models, ModelInfo{ID: m.ID, Created: m.Created})
		}
	}

	sortModels(models)
	return FetchResult{Models: models}
}

// fetchOllama fetches models from the Ollama /api/tags endpoint.
func fetchOllama(baseURL, _ string) FetchResult {
	url := strings.TrimRight(baseURL, "/") + "/api/tags"
	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return FetchResult{Err: fmt.Errorf("fetching ollama models: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return FetchResult{Err: fmt.Errorf("ollama tags endpoint returned status %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return FetchResult{Err: fmt.Errorf("reading ollama response: %w", err)}
	}

	var response struct {
		Models []struct {
			Name       string `json:"name"`
			ModifiedAt string `json:"modified_at"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return FetchResult{Err: fmt.Errorf("parsing ollama response: %w", err)}
	}

	models := make([]ModelInfo, 0, len(response.Models))
	for _, m := range response.Models {
		if m.Name != "" {
			var created int64
			if t, err := time.Parse(time.RFC3339Nano, m.ModifiedAt); err == nil {
				created = t.Unix()
			}
			models = append(models, ModelInfo{ID: m.Name, Created: created})
		}
	}

	sortModels(models)
	return FetchResult{Models: models}
}

// fetchOpenRouter fetches models from the OpenRouter public models endpoint.
func fetchOpenRouter(_ string, _ string) FetchResult {
	url := "https://openrouter.ai/api/v1/models"
	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return FetchResult{Err: fmt.Errorf("fetching openrouter models: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return FetchResult{Err: fmt.Errorf("openrouter models endpoint returned status %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2MB limit (larger response)
	if err != nil {
		return FetchResult{Err: fmt.Errorf("reading openrouter response: %w", err)}
	}

	var response struct {
		Data []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Created int64  `json:"created"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return FetchResult{Err: fmt.Errorf("parsing openrouter response: %w", err)}
	}

	models := make([]ModelInfo, 0, len(response.Data))
	for _, m := range response.Data {
		if m.ID != "" {
			models = append(models, ModelInfo{ID: m.ID, DisplayName: m.Name, Created: m.Created})
		}
	}

	sortModels(models)
	return FetchResult{Models: models}
}

// sortModels sorts by most recent first when timestamps are available,
// falling back to alphabetical by ID.
func sortModels(models []ModelInfo) {
	hasTimestamps := false
	for _, m := range models {
		if m.Created > 0 {
			hasTimestamps = true
			break
		}
	}

	sort.Slice(models, func(i, j int) bool {
		if hasTimestamps {
			if models[i].Created != models[j].Created {
				return models[i].Created > models[j].Created // newest first
			}
		}
		return models[i].ID < models[j].ID
	})
}
