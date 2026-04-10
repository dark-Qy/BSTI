# Feishu Personality Agent

Local web agent for collecting authorized Feishu data through `lark-cli` and generating an MBTI-like behavior report with the configured AIDP ModelHub LLM endpoint.

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
# Optional: preconfigure these to skip the app setup link.
LARK_APP_ID=
LARK_APP_SECRET=
```

## API

- `POST /api/sessions` creates a local session.
- `POST /api/sessions/{id}/login` returns the current Feishu link when authorization is needed. If an existing local token is valid or refreshable, the session becomes authenticated without returning a new link.
- `GET /api/sessions/{id}/status` returns session state.
- `POST /api/sessions/{id}/analyze` starts read-only collection and report generation.
- `GET /api/sessions/{id}/report` returns the local HTML report.
