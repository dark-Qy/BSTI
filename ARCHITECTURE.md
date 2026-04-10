# Architecture

The agent is a local Gin HTTP service with three boundaries:

- HTTP server and web UI: session creation, login link presentation, status polling, analysis trigger, report rendering.
- Sandbox executor: process-level isolation around `lark-cli`, with an allowlist, per-session working directory, and per-session `LARKSUITE_CLI_CONFIG_DIR`.
- Analysis pipeline: read-only Feishu collection, raw session-private logs, bounded analysis bundle, AIDP ModelHub report generation.

No Feishu write commands are part of the first version.

Session states are `created`, `login_pending`, `authenticated`, `collecting`, `analyzing`, `done`, and `failed`.

The sandbox command allowlist only permits:

- `lark-cli auth login --scope ... --no-wait`
- `lark-cli auth login --device-code ...`
- `lark-cli auth status`
- `lark-cli config init --new`
- read-only shortcuts for `im`, `docs`, `calendar`, `task`, `mail`, and `vc`

If `LARK_APP_ID` and `LARK_APP_SECRET` are present, the server writes a per-session CLI config with a file-backed app secret. If they are absent, the server runs `lark-cli config init --new`, returns the app setup URL, waits for it to complete, and then starts the normal OAuth device login.
