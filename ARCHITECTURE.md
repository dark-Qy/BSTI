# Architecture

The agent is a local Gin HTTP service with four boundaries:

- HTTP server and API surface: session creation, login link presentation, status polling, analysis trigger, photo asset serving, compatibility HTML report serving, and structured JSON report serving.
- React/Vite frontend: a built single-page app under `web/dist/` that is embedded into the binary and consumed through the session API for the landing, loading, and result flows.
- Sandbox executor: process-level isolation around `lark-cli`, with an allowlist, per-session working directory, and per-session `LARKSUITE_CLI_CONFIG_DIR`.
- Analysis pipeline: read-only Feishu collection, raw session-private logs, chat/docs domain summaries, staged BSPI persona prompt construction, provider-specific LLM chat classification through a unified `LLM_*` configuration, and local report generation.

Each session owns a private `lark-cli` config directory and keeps both app setup output and user authorization inside that sandbox. The server never seeds a new session from another session's app configuration. Only service-side `LARK_APP_ID` and `LARK_APP_SECRET` allow the login flow to skip the manual app setup step.

The frontend persists only the active `session_id` in browser `localStorage`. That browser-side value is a local session reference, not a Feishu OAuth token. Refreshing or reopening the same browser restores the existing server-side session by calling `GET /api/sessions/{id}/status`; different browsers and browser profiles stay isolated because they do not share local storage.

For BOE deployment, SCM packages the service as a Linux binary plus repo-owned build inputs under `output/`. TCE starts the packaged app through `bootstrap.sh`, which sets deployment-friendly defaults for `HTTP_ADDR` and `AGENT_DATA_DIR`, then launches the binary with embedded frontend and photo assets.

No Feishu write commands are part of the first version.

The BSPI catalog is stored locally in the repo and contains 20 canonical personas. The LLM only selects a single Top1 shorthand from that catalog; the server owns the official Chinese label, image path, one-line image, dimension metadata, and canonical persona description. If the model returns an unknown or incomplete result, the server retries once with validation feedback and otherwise fails closed.

LLM access is configured through `LLM_PROVIDER`, `LLM_API_URL`, `LLM_API_KEY`, `LLM_MODEL`, and `LLM_MAX_TOKENS`. The service currently ships two provider adapters, `modelhub` and `kimi`, behind one shared client interface. The provider is explicit user configuration rather than URL auto-detection, and the service intentionally does not enforce URL shape matching so custom gateways and compatible proxy URLs remain usable.

The structured report now includes:

- `highlight_tags`
- `behavior_vectors` with six fixed work-style signals
- `work_profile`
- `expression_fingerprint`
- `output_style`
- `knowledge_signals`
- `interaction_insights` with relationship summary, core collaborators, frequent people, and frequent chats
- `contrast_signals` for cross-domain contradictions, contrast points, or top-2 persona disambiguation
- `share_card` metadata for local poster export
- `coverage` summarizing successful and failed data domains

Within `interaction_insights`, final JSON and rendered reports intentionally hide internal Feishu identifiers such as `open_id` and `chat_id`. The `frequent_chats` list is reserved for group-chat signals and must not contain direct-message entries or direct-message analysis. The same identifier-scrubbing rule also applies to the new portrait fields and structured evidence items.

Session states are `created`, `config_pending`, `login_pending`, `authenticated`, `collecting`, `analyzing`, `done`, `auth_failed`, and `analysis_failed`.

Legacy disk sessions may still contain `failed`. The server keeps a compatibility branch for that value so older session files remain readable, but new writes use the explicit failure states above.

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

Frontend session restore behavior is intentionally shallow:

- restore the persisted `session_id` reference on startup when present
- auto-create a fresh session only when the restore request returns `404`
- keep non-404 restore failures visible so transient server errors are not mistaken for expired sessions
- expose an explicit "切换账号 / 新建会话" action for shared-browser handoff

Failure rendering is status-driven rather than inferred from the current step:

- `auth_failed` keeps the user in the authorization flow and surfaces whether the failure happened during setup/login or Feishu authorization
- `analysis_failed` returns the user to the landing flow with an explicit "分析失败" retry path that reuses the authenticated session

The sandbox command allowlist only permits:

- `lark-cli auth login --scope ... --no-wait`
- `lark-cli auth login --device-code ...`
- `lark-cli auth status`
- `lark-cli config init --new`
- read-only shortcuts for `im`, `docs`, `calendar`, `task`, `mail`, and `vc`
- additional `im` read shortcuts `+messages-search`, `+chat-messages-list`, and `+chat-search`

Chat collection is no longer one flat search-only step. The collector now:

- searches recent user-authored messages across chats by resolving the session user's open_id and passing it to `im +messages-search --sender`
- filters obvious non-work group chats with a local keyword blacklist before they enter relationship ranking
- treats those user-authored messages as anchors and enriches retained chats with bounded history reads fetched through explicit `im +chat-messages-list` page-token pagination rather than a nonexistent `--page-all` flag
- extracts compact causal-chain context around the user's own messages so relationship evidence comes from "my speech + trigger/topic anchor", not passive visibility
- uses trigger-message blocks for P2P and one-message topic anchors plus inline role markers for groups, then feeds the same cleaned context into both prompt construction and interaction summary ranking
- emits a compact chat summary for prompting and report rendering
- cleans retained chat evidence into a filtered raw `chat` payload so the final LLM prompt keeps useful conversation data without low-value noise

Document collection now runs two bounded searches:

- current-user-created docs through `creator_ids=[current user open_id]`
- recent-open docs

The merged set is still deduplicated for summary/reference purposes, while bounded `docs +fetch` body reads now target the first 10 current-user-created documents. Prompt construction keeps raw `docs` search results, raw `docs_content`, raw `task`, and a cleaned raw `chat` block with lightweight chat stats, while still treating docs content as a higher-priority long-horizon signal than chat when both are available. The final prompt also keeps `calendar`, `vc` / `vc_notes`, and `mail` / `mail_content` as JSON blocks after prompt-time field slimming removes obvious notification traffic, internal identifiers, jump links, version notices, avatars, and other prompt-irrelevant metadata. The session logs persist both the final `analysis-prompt.txt` and `analysis-domain-digests.json`, while `raw-*.jsonl` files continue to keep the original CLI responses.

If the current session profile already has a valid or refreshable user token according to `lark-cli auth status`, the session becomes authenticated without starting a new OAuth flow. The server does not reuse user tokens or app configuration from any other session. If `LARK_APP_ID` and `LARK_APP_SECRET` are present, the server writes a per-session CLI config with a file-backed app secret and starts the normal OAuth device login immediately. If they are absent, the server always runs `lark-cli config init --new`, returns the app setup URL, waits for it to complete inside the current session sandbox, and then starts the normal OAuth device login. The setup process uses a service-owned timeout context after the URL is returned, so finishing the HTTP request that delivered the setup link does not kill the in-progress CLI login flow.
