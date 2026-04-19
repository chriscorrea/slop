package format

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThinkingFilter_DetectModelType(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected ModelType
	}{
		{
			name:     "GPT-OSS harmony format",
			content:  `<|start|>assistant<|channel|>analysis<|message|>This is analysis<|end|><|channel|>final<|message|>Final answer`,
			expected: ModelGPTOss,
		},
		{
			name:     "DeepSeek R1 JSON format",
			content:  `{"thinking": "I need to think about this", "content": "Here is my answer"}`,
			expected: ModelDeepSeekR1JSON,
		},
		{
			name:     "DeepSeek R1 CLI with think tags",
			content:  "<think>Let me consider this problem</think>\nHere is the answer",
			expected: ModelDeepSeekR1CLI,
		},
		{
			name:     "DeepSeek R1 CLI with Thinking prefix",
			content:  "Thinking...\nThis is a complex problem\n\nThe answer is 42",
			expected: ModelDeepSeekR1CLI,
		},
		{
			name:     "No thinking content",
			content:  `This is just a regular response with no thinking markers`,
			expected: ModelUnknown,
		},
		{
			name:     "Invalid JSON",
			content:  `{"thinking": invalid json}`,
			expected: ModelUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewThinkingFilter(false, false)
			result := filter.DetectModelType(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestThinkingFilter_FilterGPTOss(t *testing.T) {
	tests := []struct {
		name                string
		content             string
		expectedThinking    string
		expectedFinal       string
		expectedHasThinking bool
	}{
		{
			name:                "Standard GPT-OSS format",
			content:             `<|start|>assistant<|channel|>analysis<|message|>I need to analyze this carefully<|end|><|channel|>final<|message|>The answer is 42<|end|>`,
			expectedThinking:    "I need to analyze this carefully",
			expectedFinal:       "The answer is 42",
			expectedHasThinking: true,
		},
		{
			name:                "Multiple analysis sections",
			content:             `<|start|>assistant<|channel|>analysis<|message|>First thought<|end|><|start|>assistant<|channel|>analysis<|message|>Second thought<|end|><|channel|>final<|message|>Final answer<|end|>`,
			expectedThinking:    "First thought\n\nSecond thought",
			expectedFinal:       "Final answer",
			expectedHasThinking: true,
		},
		{
			name:                "No final section",
			content:             `<|start|>assistant<|channel|>analysis<|message|>Just thinking<|end|>Some other content`,
			expectedThinking:    "Just thinking",
			expectedFinal:       "Some other content", // fallback behavior
			expectedHasThinking: true,
		},
		{
			name:                "No analysis section",
			content:             `<|channel|>final<|message|>Just the answer<|end|>`,
			expectedThinking:    "",
			expectedFinal:       "Just the answer",
			expectedHasThinking: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewThinkingFilter(false, false)
			result, err := filter.filterGPTOss(tt.content)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedThinking, result.ThinkingContent)
			assert.Equal(t, tt.expectedFinal, result.FinalContent)
			assert.Equal(t, tt.expectedHasThinking, result.HasThinking)
		})
	}
}

func TestThinkingFilter_FilterDeepSeekJSON(t *testing.T) {
	tests := []struct {
		name                string
		content             string
		expectedThinking    string
		expectedFinal       string
		expectedHasThinking bool
		expectError         bool
	}{
		{
			name:                "Valid JSON with thinking",
			content:             `{"thinking": "Let me think about this step by step", "content": "The answer is 42"}`,
			expectedThinking:    "Let me think about this step by step",
			expectedFinal:       "The answer is 42",
			expectedHasThinking: true,
			expectError:         false,
		},
		{
			name:                "Valid JSON without thinking",
			content:             `{"content": "Just the answer"}`,
			expectedThinking:    "",
			expectedFinal:       "Just the answer",
			expectedHasThinking: false,
			expectError:         false,
		},
		{
			name:                "Invalid JSON",
			content:             `{"thinking": invalid}`,
			expectedThinking:    "",
			expectedFinal:       "",
			expectedHasThinking: false,
			expectError:         true,
		},
		{
			name:                "Empty thinking field",
			content:             `{"thinking": "", "content": "The answer"}`,
			expectedThinking:    "",
			expectedFinal:       "The answer",
			expectedHasThinking: false,
			expectError:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewThinkingFilter(false, false)
			result, err := filter.filterDeepSeekJSON(tt.content)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedThinking, result.ThinkingContent)
				assert.Equal(t, tt.expectedFinal, result.FinalContent)
				assert.Equal(t, tt.expectedHasThinking, result.HasThinking)
			}
		})
	}
}

func TestThinkingFilter_FilterDeepSeekCLI(t *testing.T) {
	tests := []struct {
		name                string
		content             string
		expectedThinking    string
		expectedFinal       string
		expectedHasThinking bool
	}{
		{
			name:                "Think tags",
			content:             "<think>I need to solve this problem</think>\nThe answer is 42",
			expectedThinking:    "I need to solve this problem",
			expectedFinal:       "The answer is 42",
			expectedHasThinking: true,
		},
		{
			name:                "Multiple think tags",
			content:             "<think>First thought</think>Some text<think>Second thought</think>\nFinal answer",
			expectedThinking:    "First thought\n\nSecond thought",
			expectedFinal:       "Some text\nFinal answer",
			expectedHasThinking: true,
		},
		{
			name:                "Thinking prefix",
			content:             "Thinking...\nThis is complex\n\nThe answer is 42",
			expectedThinking:    "This is complex",
			expectedFinal:       "The answer is 42",
			expectedHasThinking: true,
		},
		{
			name:                "Mixed formats",
			content:             "<think>Tag thinking</think>\nThinking...\nPrefix thinking\n\nFinal answer here",
			expectedThinking:    "Tag thinking\n\nPrefix thinking",
			expectedFinal:       "Final answer here",
			expectedHasThinking: true,
		},
		{
			name:                "Leading newlines in think tag",
			content:             "<think>\n\nBoxer should work harder\n</think>\nFour legs good, two legs bad.",
			expectedThinking:    "Boxer should work harder",
			expectedFinal:       "Four legs good, two legs bad.",
			expectedHasThinking: true,
		},
		{
			name:                "No thinking content",
			content:             `Just a regular response`,
			expectedThinking:    "",
			expectedFinal:       "Just a regular response",
			expectedHasThinking: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewThinkingFilter(false, false)
			result, err := filter.filterDeepSeekCLI(tt.content)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedThinking, result.ThinkingContent)
			assert.Equal(t, tt.expectedFinal, result.FinalContent)
			assert.Equal(t, tt.expectedHasThinking, result.HasThinking)
		})
	}
}

func TestThinkingFilter_FormatOutput(t *testing.T) {
	tests := []struct {
		name           string
		hideThinking   bool
		showThinking   bool
		result         *FilterResult
		expectedOutput string
	}{
		{
			name:         "Hide thinking - default behavior",
			hideThinking: true,
			showThinking: false,
			result: &FilterResult{
				ThinkingContent: "Some thinking",
				FinalContent:    "Final answer",
				HasThinking:     true,
			},
			expectedOutput: "Final answer",
		},
		{
			name:         "Show thinking - formatted output",
			hideThinking: false,
			showThinking: true,
			result: &FilterResult{
				ThinkingContent: "Complex reasoning here",
				FinalContent:    "The answer is 42",
				HasThinking:     true,
			},
			expectedOutput: "Thinking:\nComplex reasoning here\n\nResponse:\nThe answer is 42",
		},
		{
			name:         "No thinking content",
			hideThinking: false,
			showThinking: true,
			result: &FilterResult{
				ThinkingContent: "",
				FinalContent:    "Just the answer",
				HasThinking:     false,
			},
			expectedOutput: "Just the answer",
		},
		{
			name:         "Both flags false - default to hide",
			hideThinking: false,
			showThinking: false,
			result: &FilterResult{
				ThinkingContent: "Some thinking",
				FinalContent:    "Final answer",
				HasThinking:     true,
			},
			expectedOutput: "Final answer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewThinkingFilter(tt.hideThinking, tt.showThinking)
			output := filter.FormatOutput(tt.result)
			assert.Equal(t, tt.expectedOutput, output)
		})
	}
}

func TestApplyThinkingFilter(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		hideThinking bool
		showThinking bool
		expected     string
	}{
		{
			name:         "GPT-OSS content with hide thinking",
			content:      `<|start|>assistant<|channel|>analysis<|message|>Analysis here<|end|><|channel|>final<|message|>Final answer<|end|>`,
			hideThinking: true,
			showThinking: false,
			expected:     "Final answer",
		},
		{
			name:         "GPT-OSS content with show thinking",
			content:      `<|start|>assistant<|channel|>analysis<|message|>Analysis here<|end|><|channel|>final<|message|>Final answer<|end|>`,
			hideThinking: false,
			showThinking: true,
			expected:     "Thinking:\nAnalysis here\n\nResponse:\nFinal answer",
		},
		{
			name:         "DeepSeek JSON with hide thinking",
			content:      `{"thinking": "Step by step reasoning", "content": "The result is 42"}`,
			hideThinking: true,
			showThinking: false,
			expected:     "The result is 42",
		},
		{
			name:         "DeepSeek CLI with show thinking",
			content:      "<think>Need to consider this</think>\nAnswer: 42",
			hideThinking: false,
			showThinking: true,
			expected:     "Thinking:\nNeed to consider this\n\nResponse:\nAnswer: 42",
		},
		{
			name:         "No thinking content",
			content:      "Just a regular response",
			hideThinking: true,
			showThinking: false,
			expected:     "Just a regular response",
		},
		{
			name:         "Complex DeepSeek with thinking prefix",
			content:      "Thinking...\nThis requires careful analysis\n\nBased on my analysis, the answer is 42",
			hideThinking: false,
			showThinking: true,
			expected:     "Thinking:\nThis requires careful analysis\n\nResponse:\nBased on my analysis, the answer is 42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ApplyThinkingFilter(tt.content, tt.hideThinking, tt.showThinking)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestThinkingFilter_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		hideThinking bool
		showThinking bool
		expected     string
	}{
		{
			name:         "Empty content",
			content:      "",
			hideThinking: true,
			showThinking: false,
			expected:     "",
		},
		{
			name:         "Whitespace only",
			content:      "   \n\t  ",
			hideThinking: true,
			showThinking: false,
			expected:     "",
		},
		{
			name:         "Malformed think tags",
			content:      "<think>Unclosed thinking",
			hideThinking: true,
			showThinking: false,
			expected:     "<think>Unclosed thinking",
		},
		{
			name:         "Nested think tags",
			content:      "<think>Outer<think>Inner</think>More outer</think>Final",
			hideThinking: true,
			showThinking: false,
			expected:     "Final",
		},
		{
			name:         "Multiple thinking prefixes",
			content:      "Thinking...\nFirst thought\n\nThinking...\nSecond thought\n\nFinal answer",
			hideThinking: false,
			showThinking: true,
			expected:     "Thinking:\nFirst thought\n\nSecond thought\n\nResponse:\nFinal answer",
		},
		{
			name:         "Very long thinking content",
			content:      "<think>" + generateLongString(1000) + "</think>\nShort answer",
			hideThinking: true,
			showThinking: false,
			expected:     "Short answer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ApplyThinkingFilter(tt.content, tt.hideThinking, tt.showThinking)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// generateLongString creates a string of specified length for testing
func generateLongString(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = 'a' + byte(i%26)
	}
	return string(result)
}

func TestThinkingFilter_RegexPatterns(t *testing.T) {
	// Test that all regex patterns compile correctly
	t.Run("All patterns compile", func(t *testing.T) {
		filter := NewThinkingFilter(false, false)

		// Test that we have all expected patterns
		expectedPatterns := []string{
			"gpt_oss_analysis",
			"gpt_oss_final",
			"harmony_cleanup",
			"deepseek_think_tags",
			"deepseek_thinking_dot",
			"extra_whitespace",
		}

		for _, pattern := range expectedPatterns {
			assert.Contains(t, filter.patterns, pattern, "Pattern %s should exist", pattern)
			assert.NotNil(t, filter.patterns[pattern], "Pattern %s should be compiled", pattern)
		}
	})

	// Test specific pattern behaviors
	t.Run("GPT-OSS analysis pattern", func(t *testing.T) {
		filter := NewThinkingFilter(false, false)
		pattern := filter.patterns["gpt_oss_analysis"]

		// Should match
		assert.True(t, pattern.MatchString(`<|start|>assistant<|channel|>analysis<|message|>content<|end|>`))

		// Should not match
		assert.False(t, pattern.MatchString(`<|start|>user<|channel|>analysis<|message|>content<|end|>`))
		assert.False(t, pattern.MatchString(`<|start|>assistant<|channel|>final<|message|>content<|end|>`))
	})

	t.Run("DeepSeek think tags pattern", func(t *testing.T) {
		filter := NewThinkingFilter(false, false)
		pattern := filter.patterns["deepseek_think_tags"]

		// Should match
		assert.True(t, pattern.MatchString(`<think>content</think>`))
		assert.True(t, pattern.MatchString(`prefix<think>multi\nline\ncontent</think>suffix`))

		// Should not match
		assert.False(t, pattern.MatchString(`<think>unclosed`))
		assert.False(t, pattern.MatchString(`closed</think>`))
	})
}
