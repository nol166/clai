# clai

**Ask your terminal anything.**

`clai` is a zero-friction CLI assistant that answers developer questions and generates shell commands — no browser, no context switching, no preamble. Just the answer.

```
$ clai "switch gcloud project"
gcloud config set project PROJECT_ID

$ clai "extract the owner field from every track.yml in this repo"
find . -name "track.yml" -exec grep -h "^owner:" {} \; | awk '{print $2}'

$ clai "alias to print the most recent mongosh log file"
alias mongolog='ls -t ~/.mongodb/mongosh/*.log 2>/dev/null | head -1 | xargs cat'

$ clai -c "one-liner to count lines changed in last commit"
git diff HEAD~1 --stat | tail -1
# copied to clipboard
```

Supports **OpenAI**, **Anthropic**, and any **OpenAI-compatible** endpoint (LiteLLM, Ollama, etc.).

---

## Install

**Go:**

```bash
go install github.com/nol166/clai@latest
```

**Binary:** grab the latest from the [Releases](https://github.com/nol166/clai/releases) page — Linux, macOS, and Windows builds available for amd64 and arm64.

---

## Quick start

Set an API key and start asking:

```bash
# OpenAI (default)
export OPENAI_API_KEY=sk-...
clai "find all TODO comments in this repo"

# Anthropic
export CLAI_PROVIDER=anthropic
export ANTHROPIC_API_KEY=sk-ant-...
clai "find all TODO comments in this repo"
```

For persistent config, run the interactive setup wizard:

```bash
clai config
```

---

## Configuration

Config is stored at `~/.config/clai/config.yaml` (respects `XDG_CONFIG_HOME`). Environment variables take precedence over the file.

| Env var | Config key | Default | Description |
|---|---|---|---|
| `CLAI_PROFILE` | `active` | `default` | Which profile to use |
| `CLAI_PROVIDER` | `provider` | `openai` | `openai`, `anthropic`, `litellm`, `ollama` |
| `CLAI_API_KEY` | `api_key` | — | API key (falls back to `OPENAI_API_KEY` / `ANTHROPIC_API_KEY`) |
| `CLAI_MODEL` | `model` | `gpt-4o-mini` | Model name |
| `CLAI_BASE_URL` | `base_url` | — | Custom API base URL (LiteLLM, Ollama, etc.) |

### Profiles

Keep independent provider/model/key combos and switch between them:

```bash
# create profiles
clai config profile add work --provider anthropic --model claude-haiku-4-5-20251001 --api-key sk-ant-...
clai config profile add local --provider ollama --model llama3.2

# switch the default
clai config profile use local

# one-shot override without switching
clai -p work "explain this stack trace"

# manage
clai config profile list
clai config profile delete local
```

All commands (`config list`, `config set`, `models list`, queries) operate on the active profile, or the one given with `-p/--profile`. An existing flat config file is migrated automatically into a `default` profile.

Profiles are stored under a `profiles` key:

```yaml
active: local
profiles:
  work:
    provider: anthropic
    api_key: sk-ant-...
    model: claude-haiku-4-5-20251001
  local:
    provider: ollama
    model: llama3.2
    base_url: http://localhost:11434/v1
```

### Non-OpenAI providers

**Anthropic:**
```bash
clai config set provider anthropic
clai config set api-key sk-ant-...
```

**Ollama (local):**
```bash
clai config set provider ollama
# base_url defaults to http://localhost:11434/v1
```

**LiteLLM:**
```bash
clai config set provider litellm
clai config set base-url http://localhost:4000/v1
clai config set model gpt-4o
```

---

## Usage

```
clai <query>              Ask anything
clai -c <query>           Ask and copy response to clipboard
clai --version            Print version
clai --help               Print help

clai config               Interactive setup wizard (edits active profile)
clai config list          Show current config
clai config set <k> <v>   Set a config value (provider, model, api-key, base-url)
clai config clipboard     Toggle auto-copy on/off

clai config profile add <name> [flags]   Create/update a profile
clai config profile use <name>           Switch the active profile
clai config profile list                 List profiles
clai config profile delete <name>        Delete a profile
clai -p <name> <query>                   Use a profile for one invocation

clai models list          List available models for the current profile
```

`clai` automatically injects your OS, shell, and current working directory into every query, so answers like *"list all .go files here"* are path-aware without any extra effort.

---

## Contributing

PRs welcome. Uses [Conventional Commits](https://www.conventionalcommits.org/) — versioning, changelogs, and release binaries are fully automated on merge to `main`.

```bash
git clone https://github.com/nol166/clai
cd clai
go build -o clai .
./clai "hello world"
```

Use `feat:` for new features, `fix:` for bug fixes.
