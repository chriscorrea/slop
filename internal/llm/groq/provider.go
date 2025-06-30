// Package groq provides a client implementation for the Groq API.
//
// API Reference: https://console.groq.com/docs/api-reference#chat
// Authentication: providers.groq.api_key or GROQ_API_KEY environment variable
//
// Example usage:
//   client := groq.NewClient(apiKey)
//   response, err := client.Generate(ctx, messages, groq.WithTemperature(0.7))
//
// Sample groq models include: llama-3.1-8b-instant, llama-3.3-70b-versatile, qwen/qwen3-32b