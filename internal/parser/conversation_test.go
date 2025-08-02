package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseJSONHistory_Valid(t *testing.T) {
	content := `[
		{"role": "user", "content": "Hello"},
		{"role": "assistant", "content": "Hi there!"},
		{"role": "user", "content": "How are you?"}
	]`
	
	messages, err := ParseJSONHistory([]byte(content))
	
	assert.NoError(t, err)
	assert.Len(t, messages, 3)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "Hello", messages[0].Content)
	assert.Equal(t, "assistant", messages[1].Role)
	assert.Equal(t, "Hi there!", messages[1].Content)
	assert.Equal(t, "user", messages[2].Role)
	assert.Equal(t, "How are you?", messages[2].Content)
}

func TestParseJSONHistory_InvalidJSON(t *testing.T) {
	content := `[
		{"role": "user", "content": "Hello",
		{"role": "assistant", "content": "Hi there!"}
	]`
	
	messages, err := ParseJSONHistory([]byte(content))
	
	assert.Error(t, err)
	assert.Nil(t, messages)
}

func TestParseJSONHistory_InvalidRole(t *testing.T) {
	content := `[
		{"role": "user", "content": "Hello"},
		{"role": "invalid", "content": "Bad role"}
	]`
	
	messages, err := ParseJSONHistory([]byte(content))
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid role 'invalid'")
	assert.Nil(t, messages)
}

func TestParseJSONHistory_EmptyContent(t *testing.T) {
	content := `[
		{"role": "user", "content": "Hello"},
		{"role": "assistant", "content": ""}
	]`
	
	messages, err := ParseJSONHistory([]byte(content))
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty message content")
	assert.Nil(t, messages)
}

func TestParseTextHistory_Valid(t *testing.T) {
	content := `User: What is the capital of France?
Assistant: The capital of France is Paris.
User: What about Italy?
Assistant: The capital of Italy is Rome.`
	
	messages, err := ParseTextHistory(content)
	
	assert.NoError(t, err)
	assert.Len(t, messages, 4)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "What is the capital of France?", messages[0].Content)
	assert.Equal(t, "assistant", messages[1].Role)
	assert.Equal(t, "The capital of France is Paris.", messages[1].Content)
	assert.Equal(t, "user", messages[2].Role)
	assert.Equal(t, "What about Italy?", messages[2].Content)
	assert.Equal(t, "assistant", messages[3].Role)
	assert.Equal(t, "The capital of Italy is Rome.", messages[3].Content)
}

func TestParseTextHistory_MarkdownFormat(t *testing.T) {
	content := `**User:** What is 2 + 2?
**Assistant:** 2 + 2 equals 4.
**User:** Thanks!`
	
	messages, err := ParseTextHistory(content)
	
	assert.NoError(t, err)
	assert.Len(t, messages, 3)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "What is 2 + 2?", messages[0].Content)
	assert.Equal(t, "assistant", messages[1].Role)
	assert.Equal(t, "2 + 2 equals 4.", messages[1].Content)
	assert.Equal(t, "user", messages[2].Role)
	assert.Equal(t, "Thanks!", messages[2].Content)
}

func TestParseTextHistory_MultilineContent(t *testing.T) {
	content := `User: Can you write a haiku?
Assistant: Here's a haiku for you:

Code flows like water
Functions dance in harmony
Bugs hide in shadows

User: Beautiful, thanks!`
	
	messages, err := ParseTextHistory(content)
	
	assert.NoError(t, err)
	assert.Len(t, messages, 3)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "Can you write a haiku?", messages[0].Content)
	assert.Equal(t, "assistant", messages[1].Role)
	assert.Contains(t, messages[1].Content, "Here's a haiku for you:")
	assert.Contains(t, messages[1].Content, "Code flows like water")
	assert.Equal(t, "user", messages[2].Role)
	assert.Equal(t, "Beautiful, thanks!", messages[2].Content)
}

func TestParseTextHistory_CaseInsensitive(t *testing.T) {
	content := `user: lowercase user
ASSISTANT: uppercase assistant
User: mixed case user`
	
	messages, err := ParseTextHistory(content)
	
	assert.NoError(t, err)
	assert.Len(t, messages, 3)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "lowercase user", messages[0].Content)
	assert.Equal(t, "assistant", messages[1].Role)
	assert.Equal(t, "uppercase assistant", messages[1].Content)
	assert.Equal(t, "user", messages[2].Role)
	assert.Equal(t, "mixed case user", messages[2].Content)
}

func TestParseTextHistory_NoMatches(t *testing.T) {
	content := `This is just regular text without conversation patterns.
It should not be parsed as a conversation.`
	
	messages, err := ParseTextHistory(content)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no conversation messages found")
	assert.Nil(t, messages)
}

func TestParseTextHistory_EmptyInput(t *testing.T) {
	content := ``
	
	messages, err := ParseTextHistory(content)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no conversation messages found")
	assert.Nil(t, messages)
}

func TestIsConversationFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"chat.conversation", true},
		{"history.chat", true},
		{"log.history", true},
		{"Chat.CONVERSATION", true}, // case insensitive
		{"file.txt", false},
		{"conversation.py", false},
		{"regular.md", false},
		{"", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsConversationFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}