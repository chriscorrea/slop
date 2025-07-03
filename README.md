# Slop

The **stochastic language operator** 
brings large language models to your command line. It’s built on the idea that language models work best as composable _language operators_.

By integrating AI directly into your shell as a pipeline-native tool, you can build sophisticated workflows for modular, observable, and effective solutions.

## Features

- **Multiple LLM providers**: Use local models ([Ollama](https://ollama.com/)) or connect to various remote providers
- **Flexible input**: Shape LLM instructions using command-line arguments, piped stdin, or text files
- **Named commands**: Configure command shortcuts with your own custom system prompts
- **Model selection**: Quickly switch between local or remote and fast or deep (reasoning) models
- **Output formatting**: Specify JSON, Markdown, YAML, or XML for structured responses 
- **Persistent project context**: Configure a directory with context files that are automatically embedded in every command
- **Configuration management**: Define all settings in a TOML config file 

## Installation

```bash
go install github.com/chriscorrea/slop@latest
```

Or build from source:

```bash
git clone https://github.com/chriscorrea/slop.git
cd slop
make build
```

### Quick Configuration

After installing, run the `init` command to configure your preferences and API keys:

```bash
slop init
```

## Basic Usage

#### Simple prompting
The most direct way to use slop is to pass your prompt as an argument.

```bash
slop "What is a large language model?"
```

#### Add files for context

The `--context` flag will prepend file content to your prompt

```bash
slop --context speech.txt "Translate this document into accessible, plain language."
```

#### Piped input

You can also pipe the output of any command directly into slop to provide context for your prompt. This is a powerful way to work with data on the fly.

```bash
curl -s https://wttr.in/ | slop "What will I need to wear for tomorrow's groundbreaking ceremony?"
```

By chaining multiple calls together, you can design sophisticated AI workflows directly in your terminal. This approach enables complex simulated reasoning, planning, and problem-solving by breaking a large task into a series of smaller, focused steps.

### Model Selection

#### Quick Inference

Use the `-f` or `--flash` flag to get a fast response from a lightweight model.

```bash
slop --flash "Which character proposed the windmill in Animal Farm?"
```

#### For Deep Reasoning

Use the `-d` or `--deep` flag for complex tasks that require reasoning models.

```bash
slop --deep "Analyze the windmill as a symbol of technological utopianism"
```

#### For Remote Processing

Use the `r` or `--remote` flag to leverage cloud-based models for enhanced capabilities.

```bash
slop --remote "Summarize the risk levels defined in the EU AI Act of 2024"
```

#### For Local Processing

Use the `-l` or `--local` flag to run models privately on your machine via Ollama.

```bash
slop --local "Elaborate on the concept of a 'Ghost in the Machine' with a 2-page report"
```

#### Supported Model Providers
Slop supports multiple LLM providers:

**Local Providers:**
- **[Ollama](https://ollama.com/)** for open-weight models including Llama, Gemma, Deepseek, and many others

**Remote Providers:**

- **[Anthropic](https://www.anthropic.com/)**
- **[Cohere](https://cohere.com/)**
- **[Groq](https://groq.com/)**
- **[Mistral AI](https://mistral.ai/)**
- **[OpenAI](https://openai.com/)**

### Persistent Context

You can set up **persistent context** that automatically includes relevant files in every slop command run from a project directory. This eliminates the need to manually specify context files.

```bash
# Set up project context by adding files
slop context add README.md

# Future slop commands in this directory will includes the file(s) as context
slop "Explain the main functionality of this project"
```

The context is managed through a `.slop/context` manifest file.

```bash
# View current project context
slop context list

# Add more files or directories
slop context add src/utils/ tests/integration/

# Clear all project context
slop context clear

# Skip project context for a single command
slop --ignore-context "Quick question without project files"
```

**Tip**: Project context files are sent as individual messages to the LLM, providing better structure and understanding compared to concatenated text.

## Configuration

Slop uses TOML configuration files located at `~/.slop/config.toml` by default.

### Basic Configuration

```toml
[providers.mistral]
api_key = "your-api-key"
base_url = "https://api.mistral.ai/v1"

[providers.ollama]
base_url = "http://localhost:11434"

[models.remote.fast]
provider = "mistral"
name = "ministral-8b-latest"

[models.remote.deep]
provider = "mistral"
name = "magistral-medium-latest"

[models.local.fast]
provider = "ollama"
name = "gemma3:latest"

[models.local.deep]
provider = "ollama"
name = "deepseek-r1:14b"

[parameters]
temperature = 0.7
max_tokens = 2048
```

### Named Commands

```toml
[commands.compress]
description = "Summarize and compress text"
system_prompt = "You are an expert at condensing information. Summarize the following text concisely while preserving key information."
model_hint = "flash"

[commands.expand]
description = "Expand and elaborate on ideas"
system_prompt = "You are an expert at expanding ideas. Take the following text and elaborate on it with detailed explanations and examples."
model_hint = "fast"
temperature = 0.75

[commands.translate]
description = "Plain language translator"
system_prompt = "You are an expert editor specializing in plain language. You will translate the user's text into clear, simple English. Break down long sentences, replace jargon with common words, and use the active voice."
model_hint = "reasoning"

[commands.review]
description = "Code reviewer"
system_prompt = "You are an expert Python programmer. Analyze the following code for bugs, improvements, and best practices."
model_hint = "reasoning"
temperature = 0.3
```

### Configuration Management

```bash
# View current configuration
slop config show

# Set configuration values
slop set providers.mistral.api_key "your-key"
slop set models.deep_remote "magistral-medium-latest"

```

### Information Commands

```bash
# List all available commands
slop list

# Show version
slop version

# Show help
slop help
slop help [command-name]
```

## Flags

### Global Flags

- `--config`: Path to config file
- `--system`: System prompt override
- `--context`: Context file paths (can be used multiple times)
- `--ignore-context`, `-i`: Ignore automated project context for this command
- `--local`, `-l`: Use local LLM provider
- `--remote`, `-r`: Use remote LLM provider  
- `--fast`, `-f`: Use fast/light model
- `--deep`, `d`: Use deep/reasoning model
- `--test`: Use mock provider for testing
- `--verbose`,  `-v`: Enable structured logging

### Output Formatting

To receive a clean, structured response, simply add one of the following flags to your command. The tool will ensure properly formatted responses by guiding the model and cleaning the raw model output. 

- `--json`: Format response as JSON
- `--jsonl`: Format response as newline-delimited JSONL
- `--yaml`: Format response as YAML
- `--md`: Format response as Markdown
- `--xml`: Format response as XML

Note: Format flags are mutually exclusive.

### Parameter Flags

- `--temperature`: Sampling randomness (higher for more creative output)
- `--max-tokens`: Maximum response length in tokens
- `--top-p`: Nucleus sampling threshold (affects variety)
- `--stop-sequences`: Stop sequences, or strings that terminate generation
