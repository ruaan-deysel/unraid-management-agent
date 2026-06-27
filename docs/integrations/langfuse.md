# Langfuse Observability

> **Status: Beta** — Opt-in. Disabled by default; no network activity unless keys
> are configured.

The Unraid Management Agent supports opt-in OpenTelemetry tracing of the embedded
autonomous Agent Core's LLM calls, tool executions, and session scoring via
[Langfuse](https://langfuse.com/). Traces are sent over OTLP/HTTP — no Langfuse
Go SDK required.

> **Related integrations:**
>
> - **Agent Core:** see the [Embedded Agent guide](agent.md) for agent configuration,
>   safety model, and autonomy settings.
> - **MCP integration:** see the [MCP guide](mcp.md) for connecting AI clients
>   directly to the agent's tools.

## Overview

When Langfuse is configured the daemon emits an OTLP trace tree for each agent
session:

```
agent-session (root)          ← langfuse.session.id attached
  └─ step-N                   ← per-iteration span
       ├─ llm-generation       ← model, token counts, masked prompt/completion
       └─ tool:<name>          ← tool call with masked args/result
```

After the session completes, three deterministic scores are submitted to the
Langfuse Scores API:

| Score name             | Meaning                                                     |
| ---------------------- | ----------------------------------------------------------- |
| `no_hallucinated_tool` | Agent called only registered tools (1 = pass, 0 = fail)     |
| `no_unconfirmed_write` | No high-risk action was executed without approval           |
| `read_only_respected`  | Read-only mode was not bypassed (emitted only when enabled) |

## Setup

### 1. Get Langfuse credentials

Sign up at [langfuse.com](https://langfuse.com/) (cloud) or self-host. From your
project settings copy:

- **Public key** (`pk-lf-…`)
- **Secret key** (`sk-lf-…`)
- **Base URL** — `https://cloud.langfuse.com` (EU) or `https://us.cloud.langfuse.com` (US)

### 2. Set environment variables

Set the following before starting the daemon:

```bash
export LANGFUSE_PUBLIC_KEY="pk-lf-..."
export LANGFUSE_SECRET_KEY="sk-lf-..."
export LANGFUSE_BASE_URL="https://us.cloud.langfuse.com"
```

Alternatively, copy `.env.example` to `.env` in the plugin directory and fill in
the values — the daemon reads `.env` on startup.

> **Plugin config:** If you use the Unraid plugin settings page, you can set these
> values under the **Langfuse** section and they are persisted to `config.cfg`.

### 3. Verify

Start (or restart) the daemon. When keys are present you will see:

```
[INFO]  Langfuse tracing enabled
```

If initialization fails (e.g. an invalid URL), the daemon logs a warning and
continues without tracing — it is always best-effort.

## What gets traced

Tracing is scoped to the embedded Agent Core. The REST API, MCP server,
collectors, and other subsystems are not traced.

### Spans

| Span name        | Parent          | Key attributes                                      |
| ---------------- | --------------- | --------------------------------------------------- |
| `agent-session`  | root            | `langfuse.session.id`, goal (truncated)             |
| `step-N`         | `agent-session` | iteration number                                    |
| `llm-generation` | `step-N`        | model name, prompt/completion token counts (masked) |
| `tool:<name>`    | `step-N`        | tool name, masked args and result                   |

All span names follow Langfuse's OTLP ingestion conventions so they appear
correctly in the Langfuse UI without additional configuration.

### Masking

Sensitive content is masked before it reaches Langfuse:

- LLM prompts and completions: retained up to a short prefix; remainder replaced
  with `[masked]`.
- Tool arguments and results: string values longer than a threshold are truncated
  and replaced with `[masked]`.

Keys (`LANGFUSE_PUBLIC_KEY`, `LANGFUSE_SECRET_KEY`) are never logged anywhere in
the daemon (`json:"-"` in config structs).

## Privacy and safety

- **Opt-in only.** No data leaves the server unless both keys are set.
- **Best-effort.** A Langfuse outage or network partition never blocks or crashes
  the agent. Spans are flushed asynchronously; any flush errors are silently
  dropped.
- **Graceful shutdown.** On daemon exit, pending spans are flushed with a 5-second
  timeout before the process terminates.
- **Works offline.** When keys are absent the tracer is a zero-cost no-op with no
  goroutines or network activity.

## Configuration reference

| Variable              | Required | Description                                                                               |
| --------------------- | -------- | ----------------------------------------------------------------------------------------- |
| `LANGFUSE_PUBLIC_KEY` | Yes      | Project public key from Langfuse settings                                                 |
| `LANGFUSE_SECRET_KEY` | Yes      | Project secret key from Langfuse settings                                                 |
| `LANGFUSE_BASE_URL`   | No       | Langfuse base URL (cloud region or self-hosted). Default: `https://us.cloud.langfuse.com` |

Tracing activates only when **both** `LANGFUSE_PUBLIC_KEY` and
`LANGFUSE_SECRET_KEY` are non-empty. `LANGFUSE_BASE_URL` defaults to
`https://cloud.langfuse.com` if omitted.

## Planned follow-on increments

- **Prompt management:** migrate system prompts to Langfuse-managed prompt
  templates for A/B testing and versioning.
- **Dataset and LLM-judge evals:** build eval datasets from traced sessions and
  run automated quality assessments.
- **Annotation queues:** surface low-scoring sessions in Langfuse's annotation UI
  for human review.

## Troubleshooting

**No traces appear in Langfuse:**

- Confirm `Langfuse tracing enabled` appears in the daemon log.
- Verify the public/secret key pair is valid for the selected base URL (EU vs US).
- Check that the Unraid server can reach `LANGFUSE_BASE_URL` on port 443.

**`Langfuse telemetry init failed` in logs:**

- The OTLP exporter rejected the endpoint URL — check `LANGFUSE_BASE_URL` ends
  with no trailing slash (e.g. `https://us.cloud.langfuse.com`).
- The daemon continues without tracing — this is not a fatal error.

**Scores not appearing:**

- Scores are submitted after a session completes. Verify the agent completed at
  least one session (check `agent.log` or the REST `/api/v1/agent/sessions`
  endpoint).
