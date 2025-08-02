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
	content := `User: What is the best pastry in all of France?
Assistant: The best pastry in France is the croissant.
User: What about Italy?
Assistant: The best pastry in Italy is the cannoli.`

	messages, err := ParseTextHistory(content)

	assert.NoError(t, err)
	assert.Len(t, messages, 4)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "What is the best pastry in all of France?", messages[0].Content)
	assert.Equal(t, "assistant", messages[1].Role)
	assert.Equal(t, "The best pastry in France is the croissant.", messages[1].Content)
	assert.Equal(t, "user", messages[2].Role)
	assert.Equal(t, "What about Italy?", messages[2].Content)
	assert.Equal(t, "assistant", messages[3].Role)
	assert.Equal(t, "The best pastry in Italy is the cannoli.", messages[3].Content)
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

sweet roots nestled deep
cinnamon whispers, softly
memories rising

User: Beautiful, thanks!`

	messages, err := ParseTextHistory(content)

	assert.NoError(t, err)
	assert.Len(t, messages, 3)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "Can you write a haiku?", messages[0].Content)
	assert.Equal(t, "assistant", messages[1].Role)
	assert.Contains(t, messages[1].Content, "Here's a haiku for you:")
	assert.Contains(t, messages[1].Content, "cinnamon whispers")
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
	content := `This is just regular content without conversation patterns.
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
