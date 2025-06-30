# LLM Architecture

This design is a work in progress, but is intended to ensure consistent behavior while allowing flexibility for each provider to handle its specific API requirements.

## Overview

```
Config → Registry.AllProviders[name] → Provider.CreateClient() → AdapterClient → Provider.{BuildRequest,ParseResponse,etc} → HTTP → LLM API
```
## Key Components

The **registry** (`internal/registry/`) is a central mapping of provider names to implemetnations. The **adapter client** (`internal/llm/common/adapter.go`) is a universal HTTP client handling retries, logging, and validation. The **provider interface** (`internal/llm/common/interfaces.go`) is a unified contract that all providers must implement.

Each **provider package** (Ollama, Mistral, and so on) implements the interface with its own `provider.go` file in a package that will also include provider-specific request/response types and generation options.

## File Structure: internal/llm/

```
internal/llm/
├── common/             # Shared interfaces and HTTP client
│   ├── adapter.go      # Universal client that uses a Provider
│   ├── base.go         # BaseClient with shared HTTP/logger config
│   ├── generation.go   # Shared generation options (temp, tokens, etc.)
│   ├── http.go         # Standardized HTTP request builders
│   ├── interfaces.go   # The core LLM and Provider interfaces
│   ├── logging.go      # Standardized logging helpers
│   ├── pointers.go     # Helpers for creating pointers for optional params
│   ├── retry.go        # Resilient request execution with backoff
│   ├── types.go        # Shared API types (Message, Usage, etc.)
│   └── validation.go   # JSON validation for structured output
├── exampleProvider/      # Specific provider implementation
│   ├── provider.go       # provider interface implementation
│   ├── client.go         # request/response types
│   └── options.go        # provider-specific options
├── .../                  # another specific provider
├── .../                  # another specific provider
└── mock/                 # mock provider for testing
```

## Request flow

1. Model selector chooses provider based on config/flags
2. `registry.CreateProvider()` instantiates the provider
3. Provider converts messages to API format via `BuildRequest()`
4. `AdapterClient` executes HTTP request
5. Provider parses response via `ParseResponse()` 
6. Errors are translated to provider-specific, actionable messages via `HandleError()`

## Required steps to add a new provider

1. **Create a package**: `internal/llm/newprovider/`
2. **Implement the provider**: All `common.Provider` methods in `provider.go`; be sure to include unit tests in `provider_test.go`
3. **Define types**: Request/response structs in `client.go`
4. **Add options**: Parameters available for the specific providerin `options.go`
5. **Register**: Add to `registry.AllProviders` map

### Other considerations for new or updated providers

- In `internal/config/config.go`, consider adding aliases for api keys (e.g. `v.RegisterAlias("newprovider-key", "providers.newprovider.api_key")` for command line and `v.BindEnv("providers.newprovider.api_key", "NEWPROVIDER_API_KEY")` for env variables)
- Add type to `/internal/config/types.go`
- Add provider defaults to `/internal/config/data/default_config.toml`
- Update `/data/configs/models/json` to inform `slop init` defaults
