package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/chriscorrea/slop/internal/llm/common"
)

var (
	// userPattern matches "User: content" or "**User:** content"
	userPattern = regexp.MustCompile(`^(?:(?i:user):\s*|\*\*(?i:user):\*\*\s*)(.*)$`)
	// assistantPattern matches "Assistant: content" or "**Assistant:** content"
	assistantPattern = regexp.MustCompile(`^(?:(?i:assistant):\s*|\*\*(?i:assistant):\*\*\s*)(.*)$`)
)

// ParseJSONHistory attempts to parse as a JSON array of messages
func ParseJSONHistory(content []byte) ([]common.Message, error) {
	var messages []common.Message
	if err := json.Unmarshal(content, &messages); err != nil {
		return nil, err
	}

	// validate message structure
	for i, msg := range messages {
		if msg.Role != "user" && msg.Role != "assistant" && msg.Role != "system" {
			return nil, fmt.Errorf("invalid role '%s' in message %d", msg.Role, i)
		}
		if strings.TrimSpace(msg.Content) == "" {
			return nil, fmt.Errorf("empty message content in message %d", i)
		}
	}

	return messages, nil
}

// ParseTextHistory attempts to parse text content as a conversation
// supports formats like "User: message" and "Assistant: message"
func ParseTextHistory(content string) ([]common.Message, error) {
	var messages []common.Message
	lines := strings.Split(content, "\n")

	var currentRole, currentContent string
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if match := userPattern.FindStringSubmatch(line); match != nil {
			// save previous message if exists
			if currentRole != "" && strings.TrimSpace(currentContent) != "" {
				messages = append(messages, common.Message{
					Role:    currentRole,
					Content: strings.TrimSpace(currentContent),
				})
			}
			currentRole = "user"
			currentContent = match[1] // captured content after "User:"
		} else if match := assistantPattern.FindStringSubmatch(line); match != nil {
			// save previous message if exists
			if currentRole != "" && strings.TrimSpace(currentContent) != "" {
				messages = append(messages, common.Message{
					Role:    currentRole,
					Content: strings.TrimSpace(currentContent),
				})
			}
			currentRole = "assistant"
			currentContent = match[1] // captured content after "Assistant:"
		} else if currentRole != "" {
			// continue building current message content
			if currentContent != "" {
				currentContent += "\n" + line
			} else {
				currentContent = line
			}
		}
		// ignore lines that don't match patterns and aren't part of an existing message
	}

	// save final message
	if currentRole != "" && strings.TrimSpace(currentContent) != "" {
		messages = append(messages, common.Message{
			Role:    currentRole,
			Content: strings.TrimSpace(currentContent),
		})
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no conversation messages found")
	}

	return messages, nil
}

// IsConversationFile checks if a file might be a conversation based on extension
func IsConversationFile(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, ".conversation") ||
		strings.HasSuffix(lower, ".chat") ||
		strings.HasSuffix(lower, ".history")
}
