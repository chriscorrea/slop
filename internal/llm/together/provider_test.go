package together

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chriscorrea/slop/internal/config"
	"github.com/chriscorrea/slop/internal/llm/common"
)

// TestProvider_CreateClient verifies client creation and configuration
func TestProvider_CreateClient(t *testing.T) {
	p := New()

	t.Run("Creates client with valid config", func(t *testing.T) {
		cfg := &config.Config{
			Providers: config.Providers{
				Together: config.Together{
					BaseProvider: config.BaseProvider{
						APIKey: "test-api-key",
					},
				},
			},
		}

		client, err := p.CreateClient(cfg, nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if client == nil {
			t.Error("Expected client to be created")
		}
	})

	t.Run("Fails with nil config", func(t *testing.T) {
		_, err := p.CreateClient(nil, nil)
		if err == nil {
			t.Error("Expected error for nil config")
		}
		if !strings.Contains(err.Error(), "config cannot be nil") {
			t.Errorf("Expected 'config cannot be nil' in error, got: %v", err)
		}
	})

	t.Run("Fails with empty API key", func(t *testing.T) {
		cfg := &config.Config{
			Providers: config.Providers{
				Together: config.Together{
					BaseProvider: config.BaseProvider{
						APIKey: "",
					},
				},
			},
		}

		_, err := p.CreateClient(cfg, nil)
		if err == nil {
			t.Error("Expected error for empty API key")
		}
		if !strings.Contains(err.Error(), "Together.AI API key is required") {
			t.Errorf("Expected 'Together.AI API key is required' in error, got: %v", err)
		}
	})

	t.Run("Respects custom base URL", func(t *testing.T) {
		cfg := &config.Config{
			Providers: config.Providers{
				Together: config.Together{
					BaseProvider: config.BaseProvider{
						APIKey:  "test-api-key",
						BaseUrl: "https://custom.together.ai",
					},
				},
			},
		}

		client, err := p.CreateClient(cfg, nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if client == nil {
			t.Error("Expected client to be created")
		}
	})
}

// TestProvider_BuildOptions verifies option building from configuration
func TestProvider_BuildOptions(t *testing.T) {
	p := New()

	t.Run("Builds options from config", func(t *testing.T) {
		cfg := &config.Config{
			Parameters: config.Parameters{
				Temperature:   0.7,
				MaxTokens:     1000,
				TopP:          0.9,
				StopSequences: []string{"END", "STOP"},
			},
			Format: config.Format{
				JSON: true,
			},
		}

		options := p.BuildOptions(cfg)
		if len(options) != 1 {
			t.Errorf("Expected 1 option, got %d", len(options))
		}

		opts, ok := options[0].(*GenerateOptions)
		if !ok {
			t.Error("Expected GenerateOptions type")
		}
		if opts.Temperature == nil || *opts.Temperature != 0.7 {
			t.Errorf("Expected temperature 0.7, got %v", opts.Temperature)
		}
		if opts.MaxTokens == nil || *opts.MaxTokens != 1000 {
			t.Errorf("Expected max tokens 1000, got %v", opts.MaxTokens)
		}
		if opts.TopP == nil || *opts.TopP != 0.9 {
			t.Errorf("Expected top_p 0.9, got %v", opts.TopP)
		}
		if len(opts.Stop) != 2 {
			t.Errorf("Expected 2 stop sequences, got %d", len(opts.Stop))
		}
		if opts.ResponseFormat == nil || opts.ResponseFormat.Type != "json_object" {
			t.Error("Expected JSON response format")
		}
	})

	t.Run("Empty config returns empty options", func(t *testing.T) {
		cfg := &config.Config{}
		options := p.BuildOptions(cfg)
		if len(options) != 1 {
			t.Errorf("Expected 1 option, got %d", len(options))
		}
		opts, ok := options[0].(*GenerateOptions)
		if !ok {
			t.Error("Expected GenerateOptions type")
		}
		if opts.Temperature != nil {
			t.Errorf("Expected no temperature, got %v", opts.Temperature)
		}
	})
}

// TestProvider_Methods verifies basic provider method returns
func TestProvider_Methods(t *testing.T) {
	p := New()

	t.Run("RequiresAPIKey returns true", func(t *testing.T) {
		if !p.RequiresAPIKey() {
			t.Error("Expected RequiresAPIKey to return true")
		}
	})

	t.Run("ProviderName returns correct name", func(t *testing.T) {
		if p.ProviderName() != "together" {
			t.Errorf("Expected provider name 'together', got %s", p.ProviderName())
		}
	})
}

// TestProvider_BuildRequest verifies request construction
func TestProvider_BuildRequest(t *testing.T) {
	p := New()

	t.Run("Builds basic request", func(t *testing.T) {
		messages := []common.Message{
			{Role: "user", Content: "Hello world"},
		}
		options := NewGenerateOptions(WithTemperature(0.7))

		req, err := p.BuildRequest(messages, "test-model", options, nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		chatReq, ok := req.(*ChatRequest)
		if !ok {
			t.Error("Expected ChatRequest type")
		}
		if chatReq.Model != "test-model" {
			t.Errorf("Expected model 'test-model', got %s", chatReq.Model)
		}
		if len(chatReq.Messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(chatReq.Messages))
		}
		if chatReq.Temperature == nil || *chatReq.Temperature != 0.7 {
			t.Errorf("Expected temperature 0.7, got %v", chatReq.Temperature)
		}
	})

	t.Run("Builds request with Together-specific options", func(t *testing.T) {
		messages := []common.Message{
			{Role: "user", Content: "Hello world"},
		}
		options := NewGenerateOptions(
			WithFrequencyPenalty(0.5),
			WithRepetitionPenalty(1.1),
			WithMinP(0.01),
			WithLogProbs(true),
		)

		req, err := p.BuildRequest(messages, "test-model", options, nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		chatReq, ok := req.(*ChatRequest)
		if !ok {
			t.Error("Expected ChatRequest type")
		}
		if chatReq.FrequencyPenalty == nil || *chatReq.FrequencyPenalty != 0.5 {
			t.Errorf("Expected frequency penalty 0.5, got %v", chatReq.FrequencyPenalty)
		}
		if chatReq.RepetitionPenalty == nil || *chatReq.RepetitionPenalty != 1.1 {
			t.Errorf("Expected repetition penalty 1.1, got %v", chatReq.RepetitionPenalty)
		}
		if chatReq.MinP == nil || *chatReq.MinP != 0.01 {
			t.Errorf("Expected min_p 0.01, got %v", chatReq.MinP)
		}
		if chatReq.LogProbs == nil || !*chatReq.LogProbs {
			t.Errorf("Expected logprobs true, got %v", chatReq.LogProbs)
		}
	})

	t.Run("Builds request with JSON format", func(t *testing.T) {
		messages := []common.Message{
			{Role: "user", Content: "Hello world"},
		}
		options := NewGenerateOptions(WithJSONFormat())

		req, err := p.BuildRequest(messages, "test-model", options, nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		chatReq, ok := req.(*ChatRequest)
		if !ok {
			t.Error("Expected ChatRequest type")
		}
		if chatReq.ResponseFormat == nil || chatReq.ResponseFormat.Type != "json_object" {
			t.Error("Expected JSON response format")
		}
	})
}

// TestProvider_ParseResponse verifies response parsing
func TestProvider_ParseResponse(t *testing.T) {
	p := New()

	t.Run("Parses valid response", func(t *testing.T) {
		resp := common.ChatResponse{
			ID:      "resp_123",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "together-model",
			Choices: []common.Choice{
				{
					Index: 0,
					Message: common.Message{
						Role:    "assistant",
						Content: "Hello! How can I help you today?",
					},
					FinishReason: "stop",
				},
			},
			Usage: common.Usage{
				PromptTokens:     10,
				CompletionTokens: 15,
				TotalTokens:      25,
			},
		}

		body, _ := json.Marshal(resp)
		content, usage, err := p.ParseResponse(body, nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if content != "Hello! How can I help you today?" {
			t.Errorf("Expected correct content, got %s", content)
		}
		if usage.TotalTokens != 25 {
			t.Errorf("Expected total tokens 25, got %d", usage.TotalTokens)
		}
	})

	t.Run("Fails with no choices", func(t *testing.T) {
		resp := common.ChatResponse{
			ID:      "resp_456",
			Choices: []common.Choice{},
		}

		body, _ := json.Marshal(resp)
		_, _, err := p.ParseResponse(body, nil)
		if err == nil {
			t.Error("Expected error for no choices")
		}
		if !strings.Contains(err.Error(), "no choices") {
			t.Errorf("Expected 'no choices' in error, got: %v", err)
		}
	})

	t.Run("Fails with invalid JSON", func(t *testing.T) {
		body := []byte(`{"invalid": json}`)
		_, _, err := p.ParseResponse(body, nil)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
		if !strings.Contains(err.Error(), "unmarshal") {
			t.Errorf("Expected 'unmarshal' in error, got: %v", err)
		}
	})
}

// TestProvider_HandleError verifies error handling
func TestProvider_HandleError(t *testing.T) {
	p := New()

	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectContains []string
	}{
		{
			name:           "401 - Authentication error",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"error": {"type": "authentication_error", "message": "Invalid API key"}}`,
			expectContains: []string{"authentication failed", "API key"},
		},
		{
			name:           "404 - Model not found",
			statusCode:     http.StatusNotFound,
			responseBody:   `{"error": {"type": "not_found", "message": "Model not found"}}`,
			expectContains: []string{"model not found"},
		},
		{
			name:           "400 - Response format error",
			statusCode:     http.StatusBadRequest,
			responseBody:   `{"error": {"type": "invalid_request_error", "param": "response_format", "message": "Invalid response_format"}}`,
			expectContains: []string{"structured output error", "response_format"},
		},
		{
			name:           "400 - Generic bad request",
			statusCode:     http.StatusBadRequest,
			responseBody:   `{"error": {"type": "invalid_request_error", "message": "Invalid request"}}`,
			expectContains: []string{"request error"},
		},
		{
			name:           "429 - Rate limit error",
			statusCode:     http.StatusTooManyRequests,
			responseBody:   `{"error": {"type": "rate_limit_error", "message": "Rate limit exceeded"}}`,
			expectContains: []string{"rate limit exceeded"},
		},
		{
			name:           "500 - Server error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"error": {"type": "internal_server_error", "message": "Internal server error"}}`,
			expectContains: []string{"server error", "temporary"},
		},
		{
			name:           "402 - Payment required",
			statusCode:     http.StatusPaymentRequired,
			responseBody:   `{"error": {"type": "payment_required"}}`,
			expectContains: []string{"insufficient credits", "payment information"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.HandleError(tt.statusCode, []byte(tt.responseBody))
			if err == nil {
				t.Error("Expected error but got none")
				return
			}

			errorStr := err.Error()
			for _, expectedFragment := range tt.expectContains {
				if !strings.Contains(strings.ToLower(errorStr), strings.ToLower(expectedFragment)) {
					t.Errorf("Expected error to contain %q, got: %v", expectedFragment, err)
				}
			}
		})
	}
}

// TestProvider_CustomizeRequest verifies request customization
func TestProvider_CustomizeRequest(t *testing.T) {
	p := New()

	t.Run("Customizes request without error", func(t *testing.T) {
		req := &http.Request{}
		err := p.CustomizeRequest(req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

// TestProvider_HandleConnectionError verifies connection error handling
func TestProvider_HandleConnectionError(t *testing.T) {
	p := New()

	t.Run("Handles connection error", func(t *testing.T) {
		originalErr := fmt.Errorf("connection refused")
		err := p.HandleConnectionError(originalErr)
		if err == nil {
			t.Error("Expected error but got none")
		}
		if !strings.Contains(err.Error(), "Failed to connect to Together.AI API") {
			t.Errorf("Expected connection error message, got: %v", err)
		}
	})
}

// TestProvider_EndToEnd provides end-to-end testing with mock server
func TestProvider_EndToEnd(t *testing.T) {
	p := New()

	t.Run("Success - Standard request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			if r.URL.Path != "/chat/completions" {
				t.Errorf("Expected /chat/completions, got %s", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer test-api-key" {
				t.Errorf("Expected Bearer token, got %s", r.Header.Get("Authorization"))
			}

			resp := common.ChatResponse{
				ID:      "resp_123",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "together-model",
				Choices: []common.Choice{
					{
						Index: 0,
						Message: common.Message{
							Role:    "assistant",
							Content: "Hello from Together.AI!",
						},
						FinishReason: "stop",
					},
				},
				Usage: common.Usage{
					PromptTokens:     10,
					CompletionTokens: 15,
					TotalTokens:      25,
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		cfg := &config.Config{
			Providers: config.Providers{
				Together: config.Together{
					BaseProvider: config.BaseProvider{
						APIKey:  "test-api-key",
						BaseUrl: server.URL,
					},
				},
			},
		}

		client, err := p.CreateClient(cfg, nil)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		messages := []common.Message{{Role: "user", Content: "Hello"}}
		options := NewGenerateOptions(WithTemperature(0.7))

		result, err := client.Generate(context.Background(), messages, "together-model", options)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if result != "Hello from Together.AI!" {
			t.Errorf("Expected correct response, got %s", result)
		}
	})

	t.Run("Success - JSON format request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req ChatRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("Failed to decode request: %v", err)
				return
			}

			if req.ResponseFormat == nil || req.ResponseFormat.Type != "json_object" {
				t.Error("Expected JSON response format")
			}

			resp := common.ChatResponse{
				ID:      "resp_json",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "together-model",
				Choices: []common.Choice{
					{
						Index: 0,
						Message: common.Message{
							Role:    "assistant",
							Content: `{"result": "success"}`,
						},
						FinishReason: "stop",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		cfg := &config.Config{
			Providers: config.Providers{
				Together: config.Together{
					BaseProvider: config.BaseProvider{
						APIKey:  "test-api-key",
						BaseUrl: server.URL,
					},
				},
			},
		}

		client, err := p.CreateClient(cfg, nil)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		messages := []common.Message{{Role: "user", Content: "Return JSON"}}
		options := NewGenerateOptions(WithJSONFormat())

		result, err := client.Generate(context.Background(), messages, "together-model", options)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		if result != `{"result": "success"}` {
			t.Errorf("Expected JSON response, got %s", result)
		}
	})

	t.Run("Error - Server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			errorResp := ErrorResponse{
				Error: ErrorDetail{
					Type:    "internal_server_error",
					Message: "Internal server error",
				},
			}
			json.NewEncoder(w).Encode(errorResp)
		}))
		defer server.Close()

		cfg := &config.Config{
			Providers: config.Providers{
				Together: config.Together{
					BaseProvider: config.BaseProvider{
						APIKey:  "test-api-key",
						BaseUrl: server.URL,
					},
				},
			},
		}

		client, err := p.CreateClient(cfg, nil)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		messages := []common.Message{{Role: "user", Content: "Hello"}}

		_, err = client.Generate(context.Background(), messages, "together-model")
		if err == nil {
			t.Error("Expected error for server error")
		}
		if !strings.Contains(err.Error(), "server error") {
			t.Errorf("Expected server error message, got: %v", err)
		}
	})
}
