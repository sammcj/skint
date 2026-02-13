package models

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchModels_OpenAICompatible(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-key")
		}
		resp := map[string]any{
			"data": []map[string]string{
				{"id": "model-b"},
				{"id": "model-a"},
				{"id": "model-c"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	result := FetchModels(srv.URL, "test-key", "some-provider")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Models) != 3 {
		t.Fatalf("got %d models, want 3", len(result.Models))
	}
	// Should be sorted alphabetically
	wantIDs := []string{"model-a", "model-b", "model-c"}
	for i, want := range wantIDs {
		if result.Models[i].ID != want {
			t.Errorf("model[%d].ID = %q, want %q", i, result.Models[i].ID, want)
		}
	}
}

func TestFetchModels_OpenAICompatible_NoAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization should be empty, got %q", got)
		}
		resp := map[string]any{
			"data": []map[string]string{{"id": "local-model"}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	result := FetchModels(srv.URL, "", "lmstudio")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Models) != 1 || result.Models[0].ID != "local-model" {
		t.Errorf("unexpected models: %v", result.Models)
	}
}

func TestFetchModels_Ollama(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		resp := map[string]any{
			"models": []map[string]any{
				{"name": "qwen3-coder:latest", "modified_at": "2025-06-15T10:30:00.123456789Z"},
				{"name": "llama3.1:latest", "modified_at": "2025-07-01T08:00:00.5Z"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	result := FetchModels(srv.URL, "", "ollama")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Models) != 2 {
		t.Fatalf("got %d models, want 2", len(result.Models))
	}
	// Sorted newest first (llama3.1 has later modified_at)
	if result.Models[0].ID != "llama3.1:latest" {
		t.Errorf("model[0].ID = %q, want %q", result.Models[0].ID, "llama3.1:latest")
	}
	if result.Models[1].ID != "qwen3-coder:latest" {
		t.Errorf("model[1].ID = %q, want %q", result.Models[1].ID, "qwen3-coder:latest")
	}
	// Verify timestamps were actually parsed
	if result.Models[0].Created == 0 {
		t.Error("expected non-zero Created timestamp for llama3.1")
	}
	if result.Models[1].Created == 0 {
		t.Error("expected non-zero Created timestamp for qwen3-coder")
	}
}

func TestFetchModels_NativeSkipped(t *testing.T) {
	result := FetchModels("", "", "native")
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
	if len(result.Models) != 0 {
		t.Errorf("expected empty models for native, got %v", result.Models)
	}
}

func TestFetchModels_AnthropicSkipped(t *testing.T) {
	result := FetchModels("", "some-key", "anthropic")
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
	if len(result.Models) != 0 {
		t.Errorf("expected empty models for anthropic, got %v", result.Models)
	}
}

func TestFetchModels_LlamaCppSilentFailure(t *testing.T) {
	// llamacpp uses the silent strategy -- errors are swallowed
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	result := FetchModels(srv.URL, "", "llamacpp")
	if result.Err != nil {
		t.Errorf("llamacpp should silently fail, got error: %v", result.Err)
	}
	if len(result.Models) != 0 {
		t.Errorf("expected empty models on failure, got %v", result.Models)
	}
}

func TestFetchModels_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	result := FetchModels(srv.URL, "bad-key", "some-provider")
	if result.Err == nil {
		t.Error("expected error for 401 response")
	}
}

func TestFetchModels_EmptyBaseURL(t *testing.T) {
	// Unknown provider with no base URL should return empty
	result := FetchModels("", "", "unknown-provider")
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
	if len(result.Models) != 0 {
		t.Errorf("expected empty models, got %v", result.Models)
	}
}

func TestFetchModels_EmptyIDsFiltered(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": []map[string]string{
				{"id": "valid-model"},
				{"id": ""},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	result := FetchModels(srv.URL, "", "minimax")
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Models) != 1 {
		t.Fatalf("expected 1 model (empty ID filtered), got %d", len(result.Models))
	}
}

func TestModelInfo_Label(t *testing.T) {
	tests := []struct {
		model ModelInfo
		want  string
	}{
		{ModelInfo{ID: "model-1"}, "model-1"},
		{ModelInfo{ID: "model-1", DisplayName: "Model One"}, "Model One"},
		{ModelInfo{ID: "x", DisplayName: ""}, "x"},
	}
	for _, tt := range tests {
		if got := tt.model.Label(); got != tt.want {
			t.Errorf("ModelInfo{ID:%q, DisplayName:%q}.Label() = %q, want %q",
				tt.model.ID, tt.model.DisplayName, got, tt.want)
		}
	}
}
