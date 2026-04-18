# clai

**Ask your terminal anything.**

`clai` is a zero-friction CLI assistant that answers developer questions and generates shell commands on the spot. No browser. No context switching. Just answers.

```bash
$ clai "what is the command to switch my gcloud project"
gcloud config set project PROJECT_ID

$ clai "extract the owner field from every track.yml in this repo"
find . -name "track.yml" -exec grep -h "^owner:" {} \; | awk '{print $2}'

$ clai "write me an alias for printing the most recent mongosh log file"
alias mongolog='ls -t ~/.mongodb/mongosh/*.log 2>/dev/null | head -1 | xargs cat'
```

Supports **OpenAI**, **Anthropic**, and any **OpenAI-compatible** endpoint (LiteLLM, Ollama, etc.).

---

## Install

**Go:**
```bash
go install github.com/johnmccambridge/clai@latest
```

**Download binary:** grab the latest from the [Releases](https://github.com/johnmccambridge/clai/releases) page — Linux, macOS, and Windows binaries available.

---

## Setup

Set your API key and you're done:

```bash
# OpenAI (default)
export OPENAI_API_KEY=sk-...

# Anthropic
export CLAI_PROVIDER=anthropic
export ANTHROPIC_API_KEY=sk-ant-...
```

Or create `~/.config/clai/config.yaml` for persistent config:

```yaml
provider: openai   # openai | anthropic | litellm
api_key: sk-...
model: gpt-4o-mini
```

---

## Configuration

| Env var | Config key | Default | Description |
|---|---|---|---|
| `CLAI_PROVIDER` | `provider` | `openai` | `openai`, `anthropic`, `litellm` |
| `CLAI_API_KEY` | `api_key` | — | API key (or use `OPENAI_API_KEY` / `ANTHROPIC_API_KEY`) |
| `CLAI_MODEL` | `model` | `gpt-4o-mini` | Model name |
| `CLAI_BASE_URL` | `base_url` | — | Custom API base URL |

### LiteLLM / Ollama

```bash
export CLAI_PROVIDER=litellm
export CLAI_BASE_URL=http://localhost:4000/v1
export CLAI_MODEL=gpt-4o
clai "how do I reverse a list in python"
```

---

## Usage

```
clai <query>
clai --version
```

`clai` injects your current working directory and shell into the system prompt automatically — so questions like *"list all .go files in this project"* get path-aware answers.

---

## Contributing

PRs welcome. Uses [Conventional Commits](https://www.conventionalcommits.org/) — versioning and changelog are fully automated.

```bash
git clone https://github.com/johnmccambridge/clai
cd clai
go mod tidy
go build -o clai .
./clai "hello world"
```

Use `feat:` for new features, `fix:` for bug fixes. Release PRs and binaries are created automatically on merge to `main`.
