# Personality Agent Design

The first version supports both service-side Feishu app credentials and automatic app setup through `lark-cli config init --new`. Each user session receives an isolated local config directory, never reuses another session's app setup, and can only execute approved `lark-cli` read commands.

The analysis window defaults to the last 30 days. Collection covers chat messages, docs, calendar events, tasks, mail, and meeting notes when permissions allow. Domain failures are treated as partial failures and are included in the final report context.

Persona visual assets use canonical shorthand based on memorable common words such as `PRISM`, `SPARK`, and `HUMBLE`. Asset filenames under `photos/` follow the pattern `<SHORTHAND>.png`.

The analysis output is a strict 20-choice BSPI classification, not a free-form label. The LLM receives the raw authorized data plus a compact local catalog of the 20 official personas, and must return one Top1 shorthand plus structured analysis fields. The server validates that shorthand against the local catalog, enriches it with the official label, dimensions, image URL, one-liner, canonical description, fixed behavior-vector schema, local share-card metadata, and data coverage summary, and then renders the result page.

LLM access uses one explicit configuration surface: `LLM_PROVIDER`, `LLM_API_URL`, `LLM_API_KEY`, `LLM_MODEL`, and `LLM_MAX_TOKENS`. The service currently supports `modelhub` and `kimi` through provider adapters behind one shared client interface. Provider choice is explicit user configuration, and the service does not validate whether `LLM_API_URL` looks like a canonical upstream URL so custom gateways and compatible proxies remain valid.

The structured analysis JSON now includes:

- `primary_persona`
- `summary`
- `evidence`
- `communication_style`
- `work_preferences`
- `blind_spots`
- `highlight_tags`
- `behavior_vectors`
- `confidence`
- `disclaimer`

`behavior_vectors` must always be a fixed 4-item array in this exact order:

1. `协作方式`: `独立成局` ↔ `高频协同`
2. `表达风格`: `克制压缩` ↔ `高频输出`
3. `决策路径`: `证据校准` ↔ `直觉快判`
4. `推进节奏`: `稳态推进` ↔ `高压突进`

For the first BOE deployment, the service stays as a Gin HTTP application rather than moving to Kitex. SCM builds a Linux binary and packages it with `photos/`, `web/dist/`, and a repo-owned `bootstrap.sh`. TCE is expected to inject the required `LLM_*` variables plus optional Feishu secrets such as `LARK_APP_ID` and `LARK_APP_SECRET` through runtime environment variables, while `bootstrap.sh` defaults the bind address to `0.0.0.0:8787` and the writable data directory to the packaged `data/` path.

The HTTP service continues to keep `GET /api/sessions/{id}/report` as a compatibility HTML fallback, while the React frontend consumes the new `GET /api/sessions/{id}/report-data` endpoint plus additive status fields `progress`, `events`, `next_action`, and `app_config_required`.

The deployed service exposes `GET /healthz` as a process-level health endpoint. It is intentionally shallow and only confirms that the HTTP server is up, which keeps it safe for liveness and readiness checks during the first single-instance BOE rollout.
