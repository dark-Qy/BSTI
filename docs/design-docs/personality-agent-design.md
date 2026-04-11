# Personality Agent Design

The first version supports both service-side Feishu app credentials and automatic app setup through `lark-cli config init --new`. Each user session receives an isolated local config directory and can only execute approved `lark-cli` read commands.

The analysis window defaults to the last 30 days. Collection covers chat messages, docs, calendar events, tasks, mail, and meeting notes when permissions allow. Domain failures are treated as partial failures and are included in the final report context.

Persona visual assets use canonical shorthand based on memorable common words such as `PRISM`, `SPARK`, and `HUMBLE`. Asset filenames under `photos/` follow the pattern `<SHORTHAND>.png`.

The analysis output is a strict 20-choice BSPI classification, not a free-form label. The LLM receives the raw authorized data plus a compact local catalog of the 20 official personas, and must return one Top1 shorthand plus structured analysis fields. The server validates that shorthand against the local catalog, enriches it with the official label, dimensions, image URL, one-liner, and canonical description, and then renders the result page.

LLM access uses one explicit configuration surface: `LLM_PROVIDER`, `LLM_API_URL`, `LLM_API_KEY`, `LLM_MODEL`, and `LLM_MAX_TOKENS`. The service currently supports `modelhub` and `kimi` through provider adapters behind one shared client interface. Provider choice is explicit user configuration, and the service does not validate whether `LLM_API_URL` looks like a canonical upstream URL so custom gateways and compatible proxies remain valid.

For the first BOE deployment, the service stays as a Gin HTTP application rather than moving to Kitex. SCM builds a Linux binary and packages it with `photos/` and a repo-owned `bootstrap.sh`. TCE is expected to inject the required `LLM_*` variables plus optional Feishu secrets such as `LARK_APP_ID` and `LARK_APP_SECRET` through runtime environment variables, while `bootstrap.sh` defaults the bind address to `0.0.0.0:8787` and the writable data directory to the packaged `data/` path.

The deployed service exposes `GET /healthz` as a process-level health endpoint. It is intentionally shallow and only confirms that the HTTP server is up, which keeps it safe for liveness and readiness checks during the first single-instance BOE rollout.
