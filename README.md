# Slop

[![Go Version](https://img.shields.io/github/go-mod/go-version/chriscorrea/slop)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/chriscorrea/slop)](https://goreportcard.com/report/github.com/chriscorrea/slop)
[![CI](https://github.com/chriscorrea/slop/actions/workflows/push.yml/badge.svg?branch=main)](https://github.com/chriscorrea/slop/actions/workflows/push.yml)
[![Latest Release](https://img.shields.io/github/v/release/chriscorrea/slop)](https://github.com/chriscorrea/slop/releases)

**Slop** (_stochastic language operator_) brings large language models to your command line as simple, composable tools. 

You can treat AI models like powerful text-processing functions and chain them together, building observable and repeatable workflows without heavy tooling.

<!--
## Demo

[![asciicast](https://asciinema.org/a/QszuTQMF339iZadU3UyQlz8ss.svg?autoplay=1&loop=1)](https://asciinema.org/a/QszuTQMF339iZadU3UyQlz8ss) -->

## ✨ Highlights

- **Run Anywhere:** Get started in seconds on macOS, Linux, or Windows with a single binary.
- **Flexible AI Models:** Seamlessly switch between local models for privacy and capable cloud models for scale.
- **Create AI Workflows:** Chain AI commands together to create multi-step workflows—no complex frameworks required.
- **Reusable Commands:** Save your most useful instructions as custom commands that you can run again and again.
- **Project Context:** Automatically include relevant project files in your prompt context.
- **Structured Output:** Format responses as clean JSON, YAML, or Markdown that works with your other tools.

## 📦 Installation

### Pre-built Binaries (Recommended)

Download the latest release for your platform from [GitHub Releases](https://github.com/chriscorrea/slop/releases).

#### macOS:

```bash
# Download and install latest release
curl -L https://github.com/chriscorrea/slop/releases/latest/download/slop-darwin-amd64.tar.gz | tar xz
sudo mv slop /usr/local/bin/
```

#### Linux:
```bash
# Download and install latest release
curl -L https://github.com/chriscorrea/slop/releases/latest/download/slop-linux-amd64.tar.gz | tar xz
sudo mv slop /usr/local/bin/
```

#### Windows:

1. [Download](https://github.com/chriscorrea/slop/releases/latest) slop-windows-amd64.zip
2. Extract the archive
3. Add slop.exe to your PATH

### Homebrew (macOS)

```bash
brew tap chriscorrea/slop
brew install slop
```

### Go Install
If you have Go installed, you can install directly from source:
```bash
go install github.com/chriscorrea/slop@latest
```

### Quick Configuration

After installing, run the `init` command to configure your preferred models and API keys:

```bash
slop init
```

## 🚀 Quick Start

#### Simple Prompting

The most direct way to use slop is to pass your prompt as an argument:

```bash
slop "What is a large language model?"
```

#### Add Files for Context

Use the --context flag to include file content in your prompt:

```bash
slop --context RFI-2025-05936.xml \
"Where is the government considering buliding new data centers?"
```

#### Piped Input

Pipe command output directly into slop for dynamic data processing. This example uses [sift](https://github.com/chriscorrea/sift) to extract content from a web site:

```bash
sift https://www.drought.gov/national | \
  slop "which states are most vulnerable to drought?"
```

Similar results can be achieved via [curl](https://github.com/curl/curl) and [pandoc](https://github.com/jgm/pandoc):
```bash
curl -sL https://www.drought.gov/national | \
  pandoc -f html -t plain | \
  slop "which states are most vulnerable to drought?"
```

Chain multiple slop commands to orchestrate multi-stage solutions:

```bash
sift https://www.drought.gov/national 
 pandoc -f html -t plain | \
 slop "Which States are most vulnerable to drought"| \
 slop --context RFI-2025-05936.xml \
 "Which proposed data centers are in areas vulnerable to drought?"
```

The results (based on [this proposal](https://www.federalregister.gov/documents/2025/04/07/2025-05936/request-for-information-on-artificial-intelligence-infrastructure-on-doe-lands#h-19)) and using gemma3 local model:
```plaintext
Based on the drought conditions you mentioned and the DOE sites listed in the RFI, several proposed data centers are located in drought-vulnerable areas:

**Western States (High Drought Risk):**
- Idaho National Laboratory (Idaho)
- National Renewable Energy Laboratory (Colorado)
- Los Alamos National Laboratory (New Mexico)
- Sandia National Laboratories (New Mexico)

**Great Plains (Moderate-High Drought Risk):**
- Pantex Plant (Texas)
- Kansas City National Security Campus (Missouri)

These sites would face significant water availability challenges for data center cooling systems, which typically require substantial water resources for operations.
```

This approach lets you decompose complex problems into focused steps, making AI workflows more modular and observable.

```bash
# analyze data and create a report
cat public_comments.csv | \
  slop "Extract all feedback with negative sentiment" | \
  slop --md "Group the feedback by theme and summarize the top 3 issues" > report.md
```

## Model Selection

Use the `--fast` or `-f` flag to get a fast response from a lightweight model.

```bash
slop --fast "Who first proposed the Animal Farm windmill?"
```

Use the `--deep` or `-d` flag for more complex tasks that require reasoning models.

```bash
slop --deep "Analyze the Animal Farm windmill as a symbol of technological utopianism""
```

Use the `--remote` or `-r` flag to leverage cloud-based models.

```bash
slop --remote "Summarize the risk levels defined in the EU AI Act of 2024"
```

Use the `--local` or `-l` flag to run models privately on your machine via Ollama.

```bash
slop --local "Elaborate on the concept of a 'Ghost in the Machine' with a 2-page report"
```

You can combine these flags (for example, `-ld`) to specify the right model for your job. 

#### Supported Model Providers

- **[Ollama](https://ollama.com/)** for local open-weight models including Llama, Gemma, Deepseek, and many others

- **[Anthropic](https://www.anthropic.com/)**
- **[Cohere](https://cohere.com/)**
- **[Groq](https://groq.com/)**
- **[Mistral AI](https://mistral.ai/)**
- **[OpenAI](https://openai.com/)**
- **[TogetherAI](https://together.ai/)**

## 🔖 Named Commands
Create your own library of commands by saving your most common instructions. This lets you build a personalized set of tools for your daily workflows.

#### Configuration
To create a new command, add a [commands.<name>] section to the `/.slop/commands.toml` file located in your home directory. For example:

```toml
[commands.review]
description = "Python code reviewer"
system_prompt = """You are an expert Python programmer. 
  Analyze the provided code and deliver a review with a focus on security, performance, and maintainability."""

message_template = """Please review this code: 
```
{input}
```
List actionable and constructive suggestions and conclude with a improved code snippet."""

model_type = "deep"
temperature = 0.3
```

#### Message Templates
Named commands support `message_template` to customize how user input is integrated into the message.

Use the `{input}` placeholder to substitute user input at the specified location in the template. If no placeholder is specified, the user input will be appended to the templated message. 

#### Usage
Once configured, you can use your named workflow by passing its name to slop. The command will automatically apply your saved configuration.

```bash
cat *.py | slop review
slop analyze "this function"
slop docs "the API endpoint"  
```

## 🗄️ Persistent Context

You can automatically add relevant files in every slop command run within a project directory. This eliminates the need to manually specify context files.

```bash
# Set up project context by adding files
slop context add README.md

# slop commands in this directory will include the context file(s)
slop "Explain the goals of this project"
```

The context is managed through a `.slop/context` manifest file in the project directory.

```bash
# View current project context
slop context list

# Add more files or directories
slop context add docs/

# Clear all project context
slop context clear

# Temporarily ignore project context with -i or --ignore-context
slop --ignore-context "Quick question without project files"
```

## 🛠️ Output Formatting

To receive a structured response, add one of the following flags to your command to automatically guide the model and clean the raw model output. 

- `--json`: Format response as JSON
- `--jsonl`: Format response as newline-delimited JSONL
- `--yaml`: Format response as YAML
- `--md`: Format response as Markdown
- `--xml`: Format response as XML

Note that format flags are mutually exclusive.

## ⚙️ Configuration

#### Command-Line Configuration

```bash
# View current configuration
slop config show

# Set configuration values
slop set providers.mistral.api_key "your-key"
slop set models.deep_remote "magistral-medium-latest"
```

#### Configuration File

Slop uses TOML configuration files located at `/.slop/config.toml` in your home directory by default.

You can add or edit model providers:

```toml
[providers.anthropic]
api_key = "your-api-key"
base_url = "https://api.anthropic.com/v1"
```

You can configure remote/local and fast/deep model preferences:

```toml
[models.local.fast]
provider = "ollama"
name = "gemma3n:latest"
```

## ℹ️  Helpful Commands

```bash
# List all available commands
slop list

# Show version
slop version

# Show help
slop help
slop help [command-name]
```

### Global Flags

- `--config`: Path to config file
- `--system`: System prompt override
- `--context`: Context file paths (can be used multiple times)
- `--ignore-context`, `-i`: Ignore automated project context for this command
- `--local`, `-l`: Use local LLM provider
- `--remote`, `-r`: Use remote LLM provider  
- `--fast`, `-f`: Use fast/light model
- `--deep`, `d`: Use deep/reasoning model
- `--verbose`,  `-v`: Show request details

### Parameter Flags

- `--temperature`: Sampling randomness (higher for more creative output)
- `--max-tokens`: Maximum response length in tokens
- `--top-p`: Nucleus sampling threshold (affects variety)
- `--stop-sequences`: Stop sequences, or strings that terminate generation

## 🤝 Contributing

Contributions and issues are welcome – please see the [issues page](https://github.com/chriscorrea/slop/issues).

## 📝 License

This project is licensed under the [BSD-3 License](LICENSE).