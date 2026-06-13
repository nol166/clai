# clai

**Ask your terminal anything.**

```
$ clai "switch gcloud project"
gcloud config set project PROJECT_ID

$ clai -c "one-liner to count lines changed in last commit"
git diff HEAD~1 --stat | tail -1
# copied to clipboard
```

No browser, no context switching, no preamble — just the answer. Your OS, shell, and working directory are injected into every query, so answers are path-aware for free.

Supports **OpenAI**, **Anthropic**, and any **OpenAI-compatible** endpoint (Ollama, LiteLLM, …).

## Install

```bash
go install github.com/nol166/clai@latest
```

Or grab a binary from [Releases](https://github.com/nol166/clai/releases) (Linux/macOS/Windows, amd64/arm64).

## Quick start

```bash
export OPENAI_API_KEY=sk-...
clai "find all TODO comments in this repo"
```

For persistent config, run the interactive wizard:

```bash
clai config
```

Full command reference: `clai --help`.

## Profiles

Keep independent provider/model/key combos and switch at will:

```bash
clai config profile add work --provider anthropic --model claude-haiku-4-5-20251001 --api-key sk-ant-...
clai config profile add local --provider ollama --model llama3.2   # no key needed
clai config profile add proxy --provider litellm --base-url http://localhost:4000/v1 --model gpt-4o

clai config profile use local      # switch the default
clai -p work "explain this trace"  # one-shot, no switch
clai config profile list
```

Every command (`config list`, `config set`, `models list`, queries) follows the active profile, or the one given with `-p/--profile`.

## Configuration

Stored at `~/.config/clai/config.yaml` (respects `XDG_CONFIG_HOME`). Env vars override the file:

| Env var | Default | Description |
|---|---|---|
| `CLAI_PROFILE` | `default` | Which profile to use |
| `CLAI_PROVIDER` | `openai` | `openai`, `anthropic`, `litellm`, `ollama` |
| `CLAI_API_KEY` | — | API key (falls back to `OPENAI_API_KEY` / `ANTHROPIC_API_KEY`) |
| `CLAI_MODEL` | `gpt-4o-mini` | Model name |
| `CLAI_BASE_URL` | — | Custom API base URL (Ollama defaults to `http://localhost:11434/v1`) |

## Contributing

PRs welcome. Uses [Conventional Commits](https://www.conventionalcommits.org/) (`feat:` / `fix:`) — versioning, changelogs, and release binaries are automated on merge to `main`.

```bash
git clone https://github.com/nol166/clai && cd clai
go build -o clai . && ./clai "hello world"
```
