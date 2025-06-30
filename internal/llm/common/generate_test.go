package common

import (
	"testing"
)

func TestNewGenerateOptions_ZeroOptions(t *testing.T) {
	opts := NewGenerateOptions()

	if opts.Temperature != nil {
		t.Errorf("Expected Temperature to be nil, got %v", opts.Temperature)
	}
	if opts.TopP != nil {
		t.Errorf("Expected TopP to be nil, got %v", opts.TopP)
	}
	if opts.MaxTokens != nil {
		t.Errorf("Expected MaxTokens to be nil, got %v", opts.MaxTokens)
	}
	if opts.Stop != nil {
		t.Errorf("Expected Stop to be nil, got %v", opts.Stop)
	}
	if opts.ResponseFormat != nil {
		t.Errorf("Expected ResponseFormat to be nil, got %v", opts.ResponseFormat)
	}
	if opts.Tools != nil {
		t.Errorf("Expected Tools to be nil, got %v", opts.Tools)
	}
	if opts.ToolChoice != nil {
		t.Errorf("Expected ToolChoice to be nil, got %v", opts.ToolChoice)
	}
}

func TestNewGenerateOptions_SingleOptions(t *testing.T) {
	tests := []struct {
		name   string
		option GenerateOption
		check  func(*GenerateOptions)
	}{
		{
			name:   "WithTemperature",
			option: WithTemperature(0.8),
			check: func(opts *GenerateOptions) {
				if opts.Temperature == nil {
					t.Errorf("Expected Temperature to be set, got nil")
					return
				}
				if *opts.Temperature != 0.8 {
					t.Errorf("Expected Temperature to be 0.8, got %v", *opts.Temperature)
				}
				// verify other fields remain nil
				if opts.TopP != nil || opts.MaxTokens != nil || opts.ResponseFormat != nil {
					t.Errorf("Expected other fields to remain nil")
				}
			},
		},
		{
			name:   "WithTopP",
			option: WithTopP(0.9),
			check: func(opts *GenerateOptions) {
				if opts.TopP == nil {
					t.Errorf("Expected TopP to be set, got nil")
					return
				}
				if *opts.TopP != 0.9 {
					t.Errorf("Expected TopP to be 0.9, got %v", *opts.TopP)
				}
				// verify other fields remain nil
				if opts.Temperature != nil || opts.MaxTokens != nil || opts.ResponseFormat != nil {
					t.Errorf("Expected other fields to remain nil")
				}
			},
		},
		{
			name:   "WithMaxTokens",
			option: WithMaxTokens(1024),
			check: func(opts *GenerateOptions) {
				if opts.MaxTokens == nil {
					t.Errorf("Expected MaxTokens to be set, got nil")
					return
				}
				if *opts.MaxTokens != 1024 {
					t.Errorf("Expected MaxTokens to be 1024, got %v", *opts.MaxTokens)
				}
				// verify other fields remain nil
				if opts.Temperature != nil || opts.TopP != nil || opts.ResponseFormat != nil {
					t.Errorf("Expected other fields to remain nil")
				}
			},
		},
		{
			name:   "WithStop",
			option: WithStop([]string{"end", "stop"}),
			check: func(opts *GenerateOptions) {
				if opts.Stop == nil {
					t.Errorf("Expected Stop to be set, got nil")
					return
				}
				if len(opts.Stop) != 2 || opts.Stop[0] != "end" || opts.Stop[1] != "stop" {
					t.Errorf("Expected Stop to be [\"end\", \"stop\"], got %v", opts.Stop)
				}
				// verify other fields remain nil
				if opts.Temperature != nil || opts.TopP != nil || opts.MaxTokens != nil {
					t.Errorf("Expected other fields to remain nil")
				}
			},
		},
		{
			name:   "WithJSONFormat",
			option: WithJSONFormat(),
			check: func(opts *GenerateOptions) {
				if opts.ResponseFormat == nil {
					t.Errorf("Expected ResponseFormat to be set, got nil")
					return
				}
				if opts.ResponseFormat.Type != "json_object" {
					t.Errorf("Expected ResponseFormat.Type to be \"json_object\", got %v", opts.ResponseFormat.Type)
				}
				// verify other fields remain nil
				if opts.Temperature != nil || opts.TopP != nil || opts.MaxTokens != nil {
					t.Errorf("Expected other fields to remain nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewGenerateOptions(tt.option)
			tt.check(opts)
		})
	}
}

func TestNewGenerateOptions_MultipleOptions(t *testing.T) {
	opts := NewGenerateOptions(
		WithTemperature(0.7),
		WithMaxTokens(1024),
		WithJSONFormat(),
		WithStop([]string{"end"}),
	)

	// verify all specified options are set
	if opts.Temperature == nil || *opts.Temperature != 0.7 {
		t.Errorf("Expected Temperature to be 0.7, got %v", opts.Temperature)
	}
	if opts.MaxTokens == nil || *opts.MaxTokens != 1024 {
		t.Errorf("Expected MaxTokens to be 1024, got %v", opts.MaxTokens)
	}
	if opts.ResponseFormat == nil || opts.ResponseFormat.Type != "json_object" {
		t.Errorf("Expected ResponseFormat to be json_object, got %v", opts.ResponseFormat)
	}
	if opts.Stop == nil || len(opts.Stop) != 1 || opts.Stop[0] != "end" {
		t.Errorf("Expected Stop to be [\"end\"], got %v", opts.Stop)
	}

	// verify unspecified options remain nil
	if opts.TopP != nil {
		t.Errorf("Expected TopP to remain nil, got %v", opts.TopP)
	}
	if opts.Tools != nil {
		t.Errorf("Expected Tools to remain nil, got %v", opts.Tools)
	}
	if opts.ToolChoice != nil {
		t.Errorf("Expected ToolChoice to remain nil, got %v", opts.ToolChoice)
	}
}

func TestNewGenerateOptions_ZeroValueVsUnset(t *testing.T) {
	tests := []struct {
		name   string
		option GenerateOption
		check  func(*GenerateOptions)
	}{
		{
			name:   "WithTemperature(0.0)",
			option: WithTemperature(0.0),
			check: func(opts *GenerateOptions) {
				if opts.Temperature == nil {
					t.Errorf("Expected Temperature to be set (not nil), got nil")
					return
				}
				if *opts.Temperature != 0.0 {
					t.Errorf("Expected Temperature to be 0.0, got %v", *opts.Temperature)
				}
			},
		},
		{
			name:   "WithTopP(0.0)",
			option: WithTopP(0.0),
			check: func(opts *GenerateOptions) {
				if opts.TopP == nil {
					t.Errorf("Expected TopP to be set (not nil), got nil")
					return
				}
				if *opts.TopP != 0.0 {
					t.Errorf("Expected TopP to be 0.0, got %v", *opts.TopP)
				}
			},
		},
		{
			name:   "WithMaxTokens(0)",
			option: WithMaxTokens(0),
			check: func(opts *GenerateOptions) {
				if opts.MaxTokens == nil {
					t.Errorf("Expected MaxTokens to be set (not nil), got nil")
					return
				}
				if *opts.MaxTokens != 0 {
					t.Errorf("Expected MaxTokens to be 0, got %v", *opts.MaxTokens)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewGenerateOptions(tt.option)
			tt.check(opts)
		})
	}
}
