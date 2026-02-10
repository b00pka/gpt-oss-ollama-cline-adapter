# GPT-OSS Ollama Cline Adapter

A Go-based reverse proxy server that bridges GPT-OSS running in Ollama with Cline's expected tool calling format.
This project is based on workaround from Reddit Thread [Making GPT-OSS 20B and CLine work together](https://old.reddit.com/r/CLine/comments/1mtcj2v/making_gptoss_20b_and_cline_work_together/).



- [GPT-OSS Ollama Cline Adapter](#gpt-oss-ollama-cline-adapter)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
  - [Environment Variables](#environment-variables)
  - [Command-Line Flags](#command-line-flags)
  - [GBNF Grammar](#gbnf-grammar)
- [Building](#building)
  - [Using Docker Compose](#using-docker-compose)

# Quick Start

+ Build & start service
  ```bash
  $ cd gpt-oss-ollama-cline-adapter
  $ docker compose up -d
  ```

+ Verify from host machine that Ollama models can be listed
  ```bash
  $ curl http://localhost:8000/models
  ...
  ```

+ Go to Cline's settings and configure parameters  
  API Provider: OpenAPI Compatible  
  URL: http://localhost:8000  
  OpenAPI Compatible API Key: <any value>  
  Model ID: gpt-oss:20b (or gpt-oss:120b)  

+ Run some Cline task to test how it works


# Configuration

## Environment Variables

| Variable                 | Default                  | Description               |
|--------------------------|--------------------------|---------------------------|
| `TARGET_BASE_URL`        | `http://ollama:11434/v1` | Ollama API endpoint       |
| `TOOL_CALL_ADAPTER_HOST` | `0.0.0.0`                | Host to listen on         |
| `TOOL_CALL_ADAPTER_PORT` | `8000`                   | Port to listen on         |
| `GRAMMAR_FILE_PATH`      | `/app/cline.gbnf`        | Path to GBNF grammar file |


## Command-Line Flags

```bash
--config <path>   Path to grammar file (.gbnf)
```
## GBNF Grammar

The adapter uses a GBNF (Grammar-Based Navigation Format) file to constrain model output. The grammar forces the model to produce properly formatted responses with:

- `<|channel|>analysis<|message|>` - Analysis phase markers
- `<|start|>assistant` - Assistant message start
- `<|channel|>final<|message|>` - Final phase markers


# Building

## Using Docker Compose

```bash
$ docker compose build
```
