package format

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ModelType represents the detected model type for thinking content filtering
type ModelType int

const (
	ModelUnknown ModelType = iota
	ModelGPTOss
	ModelDeepSeekR1JSON
	ModelDeepSeekR1CLI
)

// FilterResult contains the results of thinking content filtering
type FilterResult struct {
	ThinkingContent string
	FinalContent    string
	ModelType       ModelType
	HasThinking     bool
}

// ThinkingFilter handles detection and filtering of thinking content from various model outputs
type ThinkingFilter struct {
	hideThinking bool
	showThinking bool
	patterns     map[string]*regexp.Regexp
}

// NewThinkingFilter creates a new ThinkingFilter instance with pre-compiled regex patterns
func NewThinkingFilter(hideThinking, showThinking bool) *ThinkingFilter {
	// pre-compile regex patterns for performance
	patterns := map[string]*regexp.Regexp{
		// GPT-OSS harmony format patterns
		"gpt_oss_analysis": regexp.MustCompile(`(?s)<\|start\|>assistant<\|channel\|>analysis<\|message\|>(.*?)<\|end\|>`),
		"gpt_oss_final":    regexp.MustCompile(`(?s)<\|channel\|>final<\|message\|>(.*?)(?:<\|end\|>|$)`),
		"harmony_cleanup":  regexp.MustCompile(`<\|[^|]*\|>`),

		// DeepSeek R1 CLI patterns
		"deepseek_think_tags":   regexp.MustCompile(`(?s)<think>(.*?)</think>`),
		"deepseek_thinking_dot": regexp.MustCompile(`(?s)Thinking\.\.\.(.*?)(?:\n\n|$)`),

		// general cleanup patterns
		"extra_whitespace": regexp.MustCompile(`\n{3,}`),
	}

	return &ThinkingFilter{
		hideThinking: hideThinking,
		showThinking: showThinking,
		patterns:     patterns,
	}
}

// DetectModelType analyzes content to determine the model type and thinking format
func (tf *ThinkingFilter) DetectModelType(content string) ModelType {
	// check for GPT-OSS harmony format
	if tf.patterns["gpt_oss_analysis"].MatchString(content) {
		return ModelGPTOss
	}

	// check for DeepSeek R1 JSON format
	if tf.isDeepSeekJSON(content) {
		return ModelDeepSeekR1JSON
	}

	// check for DeepSeek R1 CLI formats
	if tf.patterns["deepseek_think_tags"].MatchString(content) ||
		tf.patterns["deepseek_thinking_dot"].MatchString(content) {
		return ModelDeepSeekR1CLI
	}

	return ModelUnknown
}

// isDeepSeekJSON checks if content is DeepSeek R1 JSON format with thinking field
func (tf *ThinkingFilter) isDeepSeekJSON(content string) bool {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return false
	}

	// check for thinking and content fields
	_, hasThinking := parsed["thinking"]
	_, hasContent := parsed["content"]
	return hasThinking && hasContent
}

// FilterContent processes content based on detected model type and returns filtered result
func (tf *ThinkingFilter) FilterContent(content string) (*FilterResult, error) {
	modelType := tf.DetectModelType(content)

	var result *FilterResult
	var err error

	switch modelType {
	case ModelGPTOss:
		result, err = tf.filterGPTOss(content)
	case ModelDeepSeekR1JSON:
		result, err = tf.filterDeepSeekJSON(content)
	case ModelDeepSeekR1CLI:
		result, err = tf.filterDeepSeekCLI(content)
	default:
		// No thinking content detected, return original content
		result = &FilterResult{
			ThinkingContent: "",
			FinalContent:    content,
			ModelType:       ModelUnknown,
			HasThinking:     false,
		}
	}

	if err != nil {
		return nil, err
	}

	result.ModelType = modelType
	return result, nil
}

// filterGPTOss handles GPT-OSS harmony format filtering
func (tf *ThinkingFilter) filterGPTOss(content string) (*FilterResult, error) {
	// extract thinking content from analysis sections using submatch to get capture group
	analysisMatches := tf.patterns["gpt_oss_analysis"].FindAllStringSubmatch(content, -1)
	var thinkingParts []string
	for _, match := range analysisMatches {
		if len(match) > 1 {
			thinking := strings.TrimSpace(match[1]) // match[1] is the captured group
			if thinking != "" {
				thinkingParts = append(thinkingParts, thinking)
			}
		}
	}

	// extract final content
	finalMatches := tf.patterns["gpt_oss_final"].FindStringSubmatch(content)
	var finalContent string
	if len(finalMatches) > 1 {
		finalContent = strings.TrimSpace(finalMatches[1])
	} else {
		// as a fallback, remove analysis sections. clean up.
		finalContent = tf.patterns["gpt_oss_analysis"].ReplaceAllString(content, "")
		finalContent = tf.patterns["harmony_cleanup"].ReplaceAllString(finalContent, "")
		finalContent = strings.TrimSpace(finalContent)
	}

	return &FilterResult{
		ThinkingContent: strings.Join(thinkingParts, "\n\n"),
		FinalContent:    finalContent,
		HasThinking:     len(thinkingParts) > 0,
	}, nil
}

// filterDeepSeekJSON handles DeepSeek R1 JSON format filtering
func (tf *ThinkingFilter) filterDeepSeekJSON(content string) (*FilterResult, error) {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse DeepSeek JSON: %w", err)
	}

	thinking, _ := parsed["thinking"].(string)
	finalContent, _ := parsed["content"].(string)

	return &FilterResult{
		ThinkingContent: thinking,
		FinalContent:    finalContent,
		HasThinking:     thinking != "",
	}, nil
}

// filterDeepSeekCLI handles DeepSeek R1 CLI format filtering.
// It extracts <think>...</think> blocks and handles any
// remaining "Thinking..." prefix sections
func (tf *ThinkingFilter) filterDeepSeekCLI(content string) (*FilterResult, error) {
	remaining, tagThinking := extractThinkTags(content)

	remaining, prefixThinking := extractThinkingPrefix(remaining)

	thinkingParts := make([]string, 0, len(tagThinking)+len(prefixThinking))
	thinkingParts = append(thinkingParts, tagThinking...)
	thinkingParts = append(thinkingParts, prefixThinking...)

	// collapse 3+ newlines to 2 and strip outer whitespace
	remaining = tf.patterns["extra_whitespace"].ReplaceAllString(remaining, "\n\n")
	remaining = strings.TrimSpace(remaining)

	return &FilterResult{
		ThinkingContent: strings.Join(thinkingParts, "\n\n"),
		FinalContent:    remaining,
		HasThinking:     len(thinkingParts) > 0,
	}, nil
}

// extractThinkTags scans for <think>...</think> blocks w/ nesting awareness
func extractThinkTags(content string) (stripped string, thinking []string) {
	const openTag = "<think>"
	const closeTag = "</think>"

	var remaining strings.Builder
	var current strings.Builder
	depth := 0

	i := 0
	for i < len(content) {
		if strings.HasPrefix(content[i:], openTag) {
			depth++
			if depth > 1 {
				current.WriteString(openTag)
			}
			i += len(openTag)
			continue
		}
		if strings.HasPrefix(content[i:], closeTag) {
			if depth == 0 {
				// stray close tag — treat as literal
				remaining.WriteString(closeTag)
				i += len(closeTag)
				continue
			}
			depth--
			if depth == 0 {
				thought := strings.TrimSpace(current.String())
				if thought != "" {
					thinking = append(thinking, thought)
				}
				current.Reset()
			} else {
				current.WriteString(closeTag)
			}
			i += len(closeTag)
			continue
		}

		if depth > 0 {
			current.WriteByte(content[i])
		} else {
			remaining.WriteByte(content[i])
		}
		i++
	}

	// unclosed outer tag — restore what we captured so the content isn't lost
	if depth > 0 {
		remaining.WriteString(openTag)
		remaining.WriteString(current.String())
	}

	return remaining.String(), thinking
}

// extractThinkingPrefix finds "Thinking..." markers in content
// final answer is whatever remains after all thinking blocks removed
func extractThinkingPrefix(content string) (stripped string, thinking []string) {
	const marker = "Thinking..."
	for {
		idx := strings.Index(content, marker)
		if idx < 0 {
			break
		}
		before := content[:idx]
		after := content[idx+len(marker):]

		end := strings.Index(after, "\n\n")
		if end < 0 {
			thought := strings.TrimSpace(after)
			if thought != "" {
				thinking = append(thinking, thought)
			}
			content = before
			break
		}

		thought := strings.TrimSpace(after[:end])
		if thought != "" {
			thinking = append(thinking, thought)
		}
		content = before + after[end+2:]
	}
	return content, thinking
}

// FormatOutput formats the final output based on thinking filter settings
func (tf *ThinkingFilter) FormatOutput(result *FilterResult) string {
	// if no thinking content (or hide-thinking is true) return only final content
	if !result.HasThinking || tf.hideThinking {
		return result.FinalContent
	}

	// if show-thinking is true and we have thinking content, format...
	if tf.showThinking && result.HasThinking {
		return fmt.Sprintf("Thinking:\n%s\n\nResponse:\n%s", result.ThinkingContent, result.FinalContent)
	}

	// as fallback/default, return final content only
	return result.FinalContent
}

// ApplyThinkingFilter is the main entry point for thinking content filtering.
func ApplyThinkingFilter(content string, hideThinking, showThinking bool) (string, error) {
	filter := NewThinkingFilter(hideThinking, showThinking)

	result, err := filter.FilterContent(content)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(filter.FormatOutput(result)), nil
}
