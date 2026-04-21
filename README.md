# Feishu Personality Agent

Local web agent for collecting authorized Feishu data through `lark-cli` and generating a BSTI Top1 persona report with the configured LLM chat endpoint. The collector now uses broader read-only chat access so it can anchor on the user's own messages, extract compact causal-chain context, filter noisy groups, and combine that chat evidence with prioritized document reads.

The current UI is a React/Vite single-page app served by the Go HTTP service. It guides users through three stages:

- Landing & Auth
- Analysis & Loading
- Result & Persona Report

## Quick Start

1. Install `lark-cli` and make sure it is available on `PATH`.
   Installation guide: [lark-cli 安装教程](https://bytedance.larkoffice.com/docx/WnHkdJQM6oGpQFxm9i7ckVdenSh)
2. Fill in `.env` with the unified `LLM_*` configuration. `LARK_APP_ID` and `LARK_APP_SECRET` are optional; if blank, every new session must first configure a Feishu app through `lark-cli config init --new` before continuing to browser authorization.
3. Install and build the frontend once:

```bash
cd web
npm install
npm run build
cd ..
```

4. Run:

```bash
go run ./cmd/agent
```

5. Open `http://127.0.0.1:8787`.

The report is not a psychological diagnosis. It is a behavior-style summary based only on the data the user explicitly authorized.

The report now uses a dual-track portrait:

- a single BSPI Top1 persona as the headline conclusion
- a structured behavior and work fingerprint layer that explains how this person works, expresses, and makes decisions

The login flow now requests one-time read-only access for chat search plus chat history reads, together with the existing docs/calendar/task/mail/vc scopes. No Feishu write capability is introduced.

Chat relationship analysis now prefers the user's own speech over passive message visibility:

- initial chat search is scoped to messages authored by the authenticated user
- obvious non-work groups are filtered by a local keyword blacklist
- retained chats are enriched with bounded history reads fetched through explicit `+chat-messages-list` page-token pagination
- P2P keeps the counterpart trigger message block before each user reply, while groups keep one recent topic anchor plus inline role markers such as `[回应他人]` and `[被@后回复]`
- interaction evidence is built from the user's message plus compact trigger/topic context, with P2P weighted ahead of groups

Each session stores and uses its own private `lark-cli` config directory. The service never reuses another session's Feishu app configuration or user authorization. The only skip path is service-side preconfiguration through `LARK_APP_ID` and `LARK_APP_SECRET`, which lets every new session start directly from browser authorization.

The browser now persists only the current local `session_id` reference in `localStorage`; it does not store the Feishu OAuth token itself. Refreshing or reopening the same browser can resume the existing session and continue analysis without reauthorizing, while different browsers or browser profiles remain isolated. If the stored session no longer exists on the server, the frontend clears that stale reference and transparently creates a new session. Shared-browser handoff is handled through the explicit "切换账号 / 新建会话" action.

## Configuration

`.env` is intentionally ignored by git. Fill in:

```dotenv
LLM_PROVIDER=modelhub
LLM_API_URL=https://aidp.bytedance.net/api/modelhub/online/v2/crawl
LLM_API_KEY=xxx
LLM_MODEL=gpt-5.4-2026-03-05
LLM_MAX_TOKENS=5000
# Optional: preconfigure these to skip the app setup link.
LARK_APP_ID=
LARK_APP_SECRET=
```

Kimi example:

```dotenv
LLM_PROVIDER=kimi
LLM_API_URL=https://api.moonshot.cn/v1/chat/completions
LLM_API_KEY=Bearer xxx
LLM_MODEL=kimi-k2.5
LLM_MAX_TOKENS=5000
LARK_APP_ID=
LARK_APP_SECRET=
```

`LLM_PROVIDER` is required and currently supports `modelhub` and `kimi`. The service does not try to infer the provider from `LLM_API_URL`, and it does not enforce a URL shape match, so custom gateways or compatible proxies are allowed as long as the selected provider adapter matches the upstream protocol.

The service also reads these optional runtime environment variables:

```dotenv
LARK_CLI_BIN=lark-cli
AGENT_DATA_DIR=/tmp/byte-agent-data
HTTP_ADDR=0.0.0.0:8787
```

## BOE Deployment

Use `build.sh` as the SCM compile script. It installs frontend dependencies, builds the React/Vite app, builds a Linux binary, copies `photos/`, and assembles a deployment-ready `output/` directory with:

- `agent`
- `bootstrap.sh`
- `photos/`
- `web/dist/`
- `data/`

Recommended SCM settings:

- compile mode: SCM compile script
- compile script path: `build.sh`
- artifact upload directory: `output`
- artifact format: TAR

For local packaging verification on a non-Linux workstation, override the target platform when running the build script, for example `GOOS=darwin GOARCH=arm64 ./build.sh`.

Recommended TCE runtime settings:

- startup script: `<deploy-path>/bootstrap.sh`
- health check: `GET /healthz` on port `8787`
- runtime env: provide the required `LLM_*` variables and any optional Feishu app credentials through TCE environment variables

`bootstrap.sh` keeps runtime defaults TCE-friendly:

- binds to `0.0.0.0:8787` when `HTTP_ADDR` is unset
- stores runtime session files under `/tmp/byte-agent-data` when `AGENT_DATA_DIR` is unset
- launches the compiled binary with embedded frontend and photo assets

The packaged service still expects `lark-cli` to be available in the runtime image or environment.

## API

- `GET /healthz` returns `200` with `{"status":"ok"}` for liveness and readiness checks.
- `POST /api/sessions` creates a local session.
- `POST /api/sessions/{id}/login` returns the current Feishu link when authorization is needed. If an existing local token is valid or refreshable, the session becomes authenticated without returning a new link.
- `GET /api/sessions/{id}/status` returns session state, progress metadata, an event timeline, the next recommended action, the additive `app_config_required` flag for the landing authorization flow, and, once ready, the Top1 BSTI persona summary used by the home page card.
- `POST /api/sessions/{id}/analyze` starts read-only collection and report generation.
- `GET /api/sessions/{id}/report` returns the compatibility HTML result page with the persona image, official persona definition, and LLM-generated analysis.
- `GET /api/sessions/{id}/report-data` returns the structured JSON report consumed by the React frontend. It includes:
  - `primary_persona`
  - `analysis`
  - `work_profile`
  - `expression_fingerprint`
  - `output_style`
  - `knowledge_signals`
  - `highlight_tags`
  - `behavior_vectors`
  - `interaction_insights`
  - `contrast_signals`
  - `share_card`
  - `coverage`
- `GET /assets/photos/{SHORTHAND}.png` serves the local persona art used by the home page and report page.

The frontend startup flow is "restore first, create as fallback":

- load the persisted `session_id` from browser `localStorage` when available
- restore with `GET /api/sessions/{id}/status`
- clear the stale browser reference and create a new session only when that restore returns `404`
- keep other restore errors visible to the user instead of silently replacing the session

Session status values now distinguish authorization failures from analysis failures:

- `auth_failed`: the authorization chain failed before analysis could start, such as app setup, login start, or Feishu authorization errors
- `analysis_failed`: authorization completed, but downstream collection or report generation failed and the user can retry analysis without reauthorizing
- legacy `failed` sessions are still accepted for compatibility and are rendered as a generic authorization-style failure until they are replaced by a new session run

`behavior_vectors` is a fixed 6-item array of work-style signals:

- `协作方式`: `独立成局` ↔ `高频协同`
- `表达风格`: `克制压缩` ↔ `高频输出`
- `决策路径`: `证据校准` ↔ `直觉快判`
- `推进节奏`: `稳态推进` ↔ `高压突进`
- `信息处理`: `深度聚焦` ↔ `广度扫描`
- `风险态度`: `防御优先` ↔ `进攻优先`

`interaction_insights` is an additive structured block:

- `relationship_summary`
- `core_collaborators`
- `frequent_people`
- `frequent_chats`

Each list item includes `display_name`, `summary`, and `evidence`.
The final JSON/HTML/Markdown outputs do not expose internal identifiers such as `open_id` or `chat_id`, and `frequent_chats` is reserved for group-chat signals rather than direct-message analysis.

`analysis.evidence` is now a structured array rather than plain strings. Each item includes:

- `domains`
- `behavior`
- `strength`
- `is_cross_domain`
- `is_distinctive`

The result page now adds three portrait modules:

- `工作画像`
- `表达指纹`
- `知识信号`

When the model finds cross-domain contradictions, notable contrast points, or the key reason for rejecting the second-closest persona candidate, it can also return `contrast_signals`.

Document collection now uses a two-source rule before body fetch:

- current-user-created docs: up to 10, filtered through `creator_ids=[current user open_id]`
- recently browsed docs: up to 20
- merged and deduplicated for indexing/reference, while bounded `docs +fetch` now reads the first 10 current-user-created documents
- the final analysis prompt keeps cleaned raw `chat`, raw `docs` / `docs_content` / `task`, while `calendar`, `vc`, and filtered `mail` / `mail_content` are injected as JSON blocks that preserve the original top-level wrapper but drop prompt-irrelevant IDs, links, version notices, avatars, and similar noisy metadata
- docs content remains the higher-priority long-horizon signal during analysis
