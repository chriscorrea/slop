// Package openai provides a client implementation for the OpenAI API.
//
// API Reference: https://platform.openai.com/docs/api-reference/chat/create
// Authentication: providers.openai.api_key or OPENAI_API_KEY environment variable
//
// Example usage:
//   client := openai.NewClient(apiKey)
//   response, err := client.Generate(ctx, messages, openai.WithTemperature(0.7))
//
// OpenAI models include: gpt-4.1-2025-04-14, o4-mini-2025-04-16, o3-2025-04-16
// OpenAI model documentation: https://platform.openai.com/docs/models