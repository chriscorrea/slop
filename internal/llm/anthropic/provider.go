// Package anthropic provides a client implementation for the AnthropicAPI.
//
// API Reference: https://docs.anthropic.com/en/api/messages
// Authentication: providers.anthropic.api_key or ANTHROPIC_API_KEY environment variable
//
// Example usage:
//   client := anthropic.NewClient(apiKey)
//   response, err := client.Generate(ctx, messages, anthropic.WithTemperature(0.7))
//
// Anthropic models include: claude-3-5-haiku-latest, claude-sonnet-4-0, claude-opus-4-0