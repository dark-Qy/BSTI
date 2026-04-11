# Feishu Personality Agent

Local web agent for collecting authorized Feishu data through `lark-cli` and generating a BSPI Top1 persona report with the configured LLM chat endpoint.

The current UI is a React/Vite single-page app served by the Go HTTP service. It guides users through three stages:

- Landing & Auth
- Analysis & Loading
- Result & Persona Report

## Quick Start

1. Install `lark-cli` and make sure it is available on `PATH`.
2. Fill in `.env` with the unified `LLM_*` configuration. `LARK_APP_ID` and `LARK_APP_SECRET` are optional; if blank and no reusable app template exists, the web login flow first asks you to configure a Feishu app through `lark-cli config init --new`.
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

The service keeps a shared app template under `data/lark-cli/` so later sessions can reuse Feishu app configuration, but each session stores and uses its own `lark-cli` user token under that session's private directory. One user's authorization is not reused by another session.

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
AGENT_DATA_DIR=./data
HTTP_ADDR=127.0.0.1:8787
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

`bootstrap.sh` keeps local development defaults untouched in Go code while making the deployed service TCE-friendly:

- binds to `0.0.0.0:8787` when `HTTP_ADDR` is unset
- stores runtime session files under `<deploy-root>/data` when `AGENT_DATA_DIR` is unset
- starts from the packaged app directory so `photos/` static assets keep working

The packaged service still expects `lark-cli` to be available in the runtime image or environment.

## API

- `GET /healthz` returns `200` with `{"status":"ok"}` for liveness and readiness checks.
- `POST /api/sessions` creates a local session.
- `POST /api/sessions/{id}/login` returns the current Feishu link when authorization is needed. If an existing local token is valid or refreshable, the session becomes authenticated without returning a new link.
- `GET /api/sessions/{id}/status` returns session state, progress metadata, an event timeline, the next recommended action, and, once ready, the Top1 BSPI persona summary used by the home page card.
- `POST /api/sessions/{id}/analyze` starts read-only collection and report generation.
- `GET /api/sessions/{id}/report` returns the compatibility HTML result page with the persona image, official persona definition, and LLM-generated analysis.
- `GET /api/sessions/{id}/report-data` returns the structured JSON report consumed by the React frontend. It includes:
  - `primary_persona`
  - `analysis`
  - `highlight_tags`
  - `behavior_vectors`
  - `share_card`
  - `coverage`
- `GET /assets/photos/{SHORTHAND}.png` serves the local persona art used by the home page and report page.

`behavior_vectors` is a fixed 4-item array of work-style signals:

- `协作方式`: `独立成局` ↔ `高频协同`
- `表达风格`: `克制压缩` ↔ `高频输出`
- `决策路径`: `证据校准` ↔ `直觉快判`
- `推进节奏`: `稳态推进` ↔ `高压突进`
