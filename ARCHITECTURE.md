# Architecture

The agent is a local Gin HTTP service with four boundaries:

- HTTP server and API surface: session creation, login link presentation, status polling, analysis trigger, photo asset serving, compatibility HTML report serving, and structured JSON report serving.
- React/Vite frontend: a built single-page app under `web/dist/` that consumes the session API and renders the landing, loading, and result flows.
- Sandbox executor: process-level isolation around `lark-cli`, with an allowlist, per-session working directory, and per-session `LARKSUITE_CLI_CONFIG_DIR`.
- Analysis pipeline: read-only Feishu collection, raw session-private logs, BSPI persona prompt construction, provider-specific LLM chat classification through a unified `LLM_*` configuration, and local report generation.

Each session owns a private `lark-cli` config directory and keeps both app setup output and user authorization inside that sandbox. The server never seeds a new session from another session's app configuration. Only service-side `LARK_APP_ID` and `LARK_APP_SECRET` allow the login flow to skip the manual app setup step.

For BOE deployment, SCM packages the service as a Linux binary plus repo-owned runtime assets under `output/`. TCE starts the packaged app through `bootstrap.sh`, which sets deployment-friendly defaults for `HTTP_ADDR` and `AGENT_DATA_DIR`, then launches the binary from the packaged root so relative static assets such as `photos/` and `web/dist/` continue to resolve correctly.

No Feishu write commands are part of the first version.

The BSPI catalog is stored locally in the repo and contains 20 canonical personas. The LLM only selects a single Top1 shorthand from that catalog; the server owns the official Chinese label, image path, one-line image, dimension metadata, and canonical persona description. If the model returns an unknown or incomplete result, the server retries once with validation feedback and otherwise fails closed.

LLM access is configured through `LLM_PROVIDER`, `LLM_API_URL`, `LLM_API_KEY`, `LLM_MODEL`, and `LLM_MAX_TOKENS`. The service currently ships two provider adapters, `modelhub` and `kimi`, behind one shared client interface. The provider is explicit user configuration rather than URL auto-detection, and the service intentionally does not enforce URL shape matching so custom gateways and compatible proxy URLs remain usable.

The structured report now includes:

- `highlight_tags`
- `behavior_vectors` with four fixed work-style signals
- `share_card` metadata for local poster export
- `coverage` summarizing successful and failed data domains

Session states are `created`, `config_pending`, `login_pending`, `authenticated`, `collecting`, `analyzing`, `done`, and `failed`.

Each session also stores an append-only event timeline. Every major state transition records:

- `stage`
- `label`
- `percent`
- `timestamp`

The HTTP surface includes a lightweight `GET /healthz` endpoint that returns `200` with `{"status":"ok"}`. It is intended for TCE liveness and readiness checks and does not depend on Feishu credentials or session state.

The HTTP surface now includes:

- `GET /api/sessions/{id}/status` with additive `progress`, `events`, `next_action`, and `app_config_required` fields
- `GET /api/sessions/{id}/report-data` for the React frontend
- `GET /api/sessions/{id}/report` as a compatibility HTML fallback

The sandbox command allowlist only permits:

- `lark-cli auth login --scope ... --no-wait`
- `lark-cli auth login --device-code ...`
- `lark-cli auth status`
- `lark-cli config init --new`
- read-only shortcuts for `im`, `docs`, `calendar`, `task`, `mail`, and `vc`

If the current session profile already has a valid or refreshable user token according to `lark-cli auth status`, the session becomes authenticated without starting a new OAuth flow. The server does not reuse user tokens or app configuration from any other session. If `LARK_APP_ID` and `LARK_APP_SECRET` are present, the server writes a per-session CLI config with a file-backed app secret and starts the normal OAuth device login immediately. If they are absent, the server always runs `lark-cli config init --new`, returns the app setup URL, waits for it to complete inside the current session sandbox, and then starts the normal OAuth device login. The setup process uses a service-owned timeout context after the URL is returned, so finishing the HTTP request that delivered the setup link does not kill the in-progress CLI login flow.
