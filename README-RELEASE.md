# LlamaOfFate Release

Welcome. This archive includes pre-built binaries and starter config files for your platform.

## Included Files

- `llamaoffate-cli` or `llamaoffate-cli.exe`
- `llamaoffate-server` or `llamaoffate-server.exe`
- `configs/azure-llm.yaml`
- `configs/ollama-llm.yaml`
- `LICENSE`
- `SECURITY.md`

## Quick Start

Choose one backend:

### Option 1: Ollama

1. Install [Ollama](https://ollama.ai).
2. Pull a model such as `llama3.2:3b`.
3. Start Ollama with `ollama serve`.
4. Run LlamaOfFate with the bundled Ollama config.

macOS/Linux:

```bash
LLM_CONFIG=configs/ollama-llm.yaml ./llamaoffate-cli
```

Windows PowerShell:

```powershell
$env:LLM_CONFIG = "configs/ollama-llm.yaml"
.\llamaoffate-cli.exe
```

The bundled Ollama config expects:

```yaml
api_endpoint: "http://localhost:11434/v1/chat/completions"
api_key: "ollama"
model_name: "llama3.2:3b"
timeout: 300
```

### Option 2: Azure OpenAI-Compatible Endpoint

Set your endpoint and key with environment variables and use the bundled Azure config template.

macOS/Linux:

```bash
export AZURE_API_ENDPOINT="https://your-resource.cognitiveservices.azure.com/openai/deployments/your-deployment/chat/completions?api-version=2024-05-01-preview"
export AZURE_API_KEY="your-api-key-here"
./llamaoffate-cli
```

Windows PowerShell:

```powershell
$env:AZURE_API_ENDPOINT = "https://your-resource.cognitiveservices.azure.com/openai/deployments/your-deployment/chat/completions?api-version=2024-05-01-preview"
$env:AZURE_API_KEY = "your-api-key-here"
.\llamaoffate-cli.exe
```

The bundled `configs/azure-llm.yaml` uses this schema:

```yaml
api_endpoint: ""
api_key: ""
model_name: "Llama-4-Maverick-17B-128E-Instruct-FP8"
timeout: 300
```

Environment variables override file values.

## Config File Lookup

If `LLM_CONFIG` is a relative path, LlamaOfFate looks in this order:

1. Current working directory
2. Directory containing the executable
3. OS user config directory

User config directory locations:

- Linux/macOS: `~/.config/LlamaOfFate/`
- Windows: `%AppData%\LlamaOfFate\`

This means you can either:

- run the binaries from the extracted archive directory
- keep the `configs/` directory next to the executable
- place your config under the OS user config directory

You can always override the path explicitly:

macOS/Linux:

```bash
LLM_CONFIG=/path/to/my-llm.yaml ./llamaoffate-cli
```

Windows PowerShell:

```powershell
$env:LLM_CONFIG = "C:\path\to\my-llm.yaml"
.\llamaoffate-cli.exe
```

## Running the Server

macOS/Linux:

```bash
./llamaoffate-server
```

Windows:

```powershell
.\llamaoffate-server.exe
```

Then open http://localhost:8080 in your browser.

## Running the CLI

macOS/Linux:

```bash
./llamaoffate-cli
```

Windows:

```powershell
.\llamaoffate-cli.exe
```

## Environment Variables

- `AZURE_API_ENDPOINT`: Overrides `api_endpoint`
- `AZURE_API_KEY`: Overrides `api_key`
- `LLM_CONFIG`: Overrides the config file path
- `PORT`: Server port for `llamaoffate-server` (default `8080`)

## Troubleshooting

**LLM config not found**

- Keep `configs/` next to the executable, or set `LLM_CONFIG` explicitly.
- If you launch the binary from another directory, use an absolute `LLM_CONFIG` path.
- You can also place configs under `~/.config/LlamaOfFate/` or `%AppData%\LlamaOfFate\`.

**api_endpoint is empty / api_key is empty**

- Set `AZURE_API_ENDPOINT` and `AZURE_API_KEY`, or fill those values into your config file.
- For Ollama, use `configs/ollama-llm.yaml`.

**Connection refused**

- Verify your configured LLM endpoint is reachable.
- If using Ollama, start it with `ollama serve`.

**Binary not executable on macOS/Linux**

```bash
chmod +x llamaoffate-cli llamaoffate-server
```

## Documentation

- Full project README: https://github.com/C-Ross/LlamaOfFate
- Fate Core SRD: https://fate-srd.com/fate-core
- Issue tracker: https://github.com/C-Ross/LlamaOfFate/issues
