# LlamaOfFate Release

Welcome! This release includes pre-built binaries for your platform.

## Prerequisites

You'll need to configure an LLM provider. Choose one:

### Option 1: Ollama (Recommended for getting started)

1. [Install Ollama](https://ollama.ai)
2. Pull a model: `ollama pull mistral` or `ollama pull neural-chat`
3. Start Ollama: `ollama serve` (runs on `http://localhost:11434`)

### Option 2: Azure OpenAI (Enterprise)

1. Get your Azure endpoint and API key from Azure Portal
2. Create `configs/azure-llm.yaml`:
   ```yaml
   provider: azure
   model: gpt-4
   endpoint: https://your-resource.openai.azure.com/
   api_key: your-api-key
   ```

### Option 3: OpenAI Direct

1. Get your OpenAI API key from [platform.openai.com](https://platform.openai.com/api-keys)
2. Create `configs/azure-llm.yaml`:
   ```yaml
   provider: openai
   model: gpt-4-turbo
   api_key: sk-...
   ```

## Running the Server (Web UI)

### macOS/Linux
```bash
./llamaoffate-server
```

Open http://localhost:8080 in your browser.

### Windows
```cmd
llamaoffate-server.exe
```

Open http://localhost:8080 in your browser.

## Running the CLI (Terminal)

### macOS/Linux
```bash
./llamaoffate-cli
```

### Windows
```cmd
llamaoffate-cli.exe
```

## Configuration

- **Default config:** `configs/azure-llm.yaml`
- **Override config:** Set `LLM_CONFIG=my-config.yaml` before running

### Environment Variables
- `PORT` — Server port (default: 8080)
- `LLM_CONFIG` — Path to LLM config file (default: `configs/azure-llm.yaml`)
- `LLM_PROVIDER` — Override LLM provider (ollama, openai, azure)

## Example: Using Ollama

```bash
# Terminal 1: Start Ollama
ollama serve

# Terminal 2: Run with Ollama config
LLM_CONFIG=configs/ollama-llm.yaml ./llamaoffate-server
```

## Troubleshooting

**"Connection refused" on startup**
- Verify your LLM provider is running
- Check the configured endpoint and API key
- Try `ollama serve` if using local Ollama

**Binary not executable (macOS/Linux)**
```bash
chmod +x llamaoffate-cli llamaoffate-server
```

## Documentation

- [Full README](https://github.com/C-Ross/LlamaOfFate)
- [Fate Core Rules](https://fate-srd.com/fate-core)
- [Troubleshooting Guide](https://github.com/C-Ross/LlamaOfFate/discussions)

## Reporting Issues

Found a bug? [Create an issue](https://github.com/C-Ross/LlamaOfFate/issues) with:
- Your platform (macOS/Linux/Windows)
- Binary filename you're running
- Steps to reproduce
- LLM provider you're using
