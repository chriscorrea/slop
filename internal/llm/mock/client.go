package mock

import (
	"context"

	"slop/internal/llm/common"
)

// Client implements the common.LLM interface for mock testing
type Client struct{}

var _ common.LLM = (*Client)(nil)

// Generate implements common.LLM interface, returns a mock response
func (c *Client) Generate(ctx context.Context, messages []common.Message, modelName string, options ...interface{}) (string, error) {
	return "Mock LLM response", nil
}
