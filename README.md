# Feishu Personality Agent

Local web agent for collecting authorized Feishu data through `lark-cli` and generating a BSPI Top1 persona report with the configured AIDP ModelHub LLM endpoint.

## Quick Start

1. Install `lark-cli` and make sure it is available on `PATH`.
2. Fill in `.env` with `AIDP_AK`. `LARK_APP_ID` and `LARK_APP_SECRET` are optional; if blank and no reusable local profile exists, the web login flow first asks you to configure a Feishu app through `lark-cli config init --new`.
3. Run:

```bash
go run ./cmd/agent
```

4. Open `http://127.0.0.1:8787`.

The report is not a psychological diagnosis. It is a behavior-style summary based only on the data the user explicitly authorized.

After the first successful login, the agent reuses the local `lark-cli` profile metadata under `data/lark-cli/` and the token managed by `lark-cli`. A valid or refreshable token skips the browser authorization step on later sessions.

## Configuration

`.env` is intentionally ignored by git. Fill in:

```dotenv
AIDP_AK=xxx
AIDP_MODELHUB_URL=https://aidp.bytedance.net/api/modelhub/online/v2/crawl
AIDP_MODEL=gpt-5.4-2026-03-05
AIDP_MAX_TOKENS=5000
# Optional: preconfigure these to skip the app setup link.
LARK_APP_ID=
LARK_APP_SECRET=
```

The service also reads these optional runtime environment variables:

```dotenv
LARK_CLI_BIN=lark-cli
AGENT_DATA_DIR=./data
HTTP_ADDR=127.0.0.1:8787
```

## BOE Deployment

Use `build.sh` as the SCM compile script. It builds a Linux binary, copies `photos/`, and assembles a deployment-ready `output/` directory with:

- `agent`
- `bootstrap.sh`
- `photos/`
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
- runtime env: provide `AIDP_AK` and any optional Feishu app credentials through TCE environment variables

`bootstrap.sh` keeps local development defaults untouched in Go code while making the deployed service TCE-friendly:

- binds to `0.0.0.0:8787` when `HTTP_ADDR` is unset
- stores runtime session files under `<deploy-root>/data` when `AGENT_DATA_DIR` is unset
- starts from the packaged app directory so `photos/` static assets keep working

The packaged service still expects `lark-cli` to be available in the runtime image or environment.

## API

- `GET /healthz` returns `200` with `{"status":"ok"}` for liveness and readiness checks.
- `POST /api/sessions` creates a local session.
- `POST /api/sessions/{id}/login` returns the current Feishu link when authorization is needed. If an existing local token is valid or refreshable, the session becomes authenticated without returning a new link.
- `GET /api/sessions/{id}/status` returns session state and, once ready, the Top1 BSPI persona summary used by the home page card.
- `POST /api/sessions/{id}/analyze` starts read-only collection and report generation.
- `GET /api/sessions/{id}/report` returns the local HTML result page with the persona image, official persona definition, and LLM-generated analysis.
- `GET /assets/photos/{SHORTHAND}.png` serves the local persona art used by the home page and report page.
