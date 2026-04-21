# Personality Agent Design

The first version supports both service-side Feishu app credentials and automatic app setup through `lark-cli config init --new`. Each user session receives an isolated local config directory, never reuses another session's app setup, and can only execute approved `lark-cli` read commands.

The browser persists only the current `session_id` reference in `localStorage`, not the Feishu OAuth token itself. On startup, the React app first tries to restore that server-side session through `GET /api/sessions/{id}/status`; if the session file is gone and the request returns `404`, the frontend clears the stale browser reference and creates a new session. Other restore failures remain user-visible so backend faults are not silently converted into session resets. Shared-browser turnover is handled through an explicit "切换账号 / 新建会话" action rather than automatic account detection.

Session failure states are intentionally split by responsibility boundary. Authorization-chain failures write `auth_failed`; downstream collection, LLM validation, and report-generation failures write `analysis_failed`. Historical session files may still contain the legacy `failed` value, so the status view layer keeps a compatibility mapping for reads while all new writes use the explicit states.

The analysis window defaults to the last 30 days. Collection covers chat messages, bounded direct-message history, bounded group history, prioritized docs, calendar events, tasks, mail, and meeting notes when permissions allow. Chat extraction now converts that history into compact causal chains rather than fixed-size windows, and domain failures are treated as partial failures and are included in the final report context.

Persona visual assets use canonical shorthand based on memorable common words such as `PRISM`, `SPARK`, and `HUMBLE`. Asset filenames under `photos/` follow the pattern `<SHORTHAND>.png`.

The analysis output is a strict 20-choice BSPI classification, not a free-form label. The LLM receives the raw authorized data plus a compact local catalog of the 20 official personas, and must return one Top1 shorthand plus structured analysis fields. The server validates that shorthand against the local catalog, enriches it with the official label, dimensions, image URL, one-liner, canonical description, fixed behavior-vector schema, structured portrait fields, local share-card metadata, and data coverage summary, and then renders the result page.

LLM access uses one explicit configuration surface: `LLM_PROVIDER`, `LLM_API_URL`, `LLM_API_KEY`, `LLM_MODEL`, and `LLM_MAX_TOKENS`. The service currently supports `modelhub` and `kimi` through provider adapters behind one shared client interface. Provider choice is explicit user configuration, and the service does not validate whether `LLM_API_URL` looks like a canonical upstream URL so custom gateways and compatible proxies remain valid.

The structured analysis JSON now includes:

- `primary_persona`
- `summary`
- `evidence`
- `work_profile`
- `expression_fingerprint`
- `output_style`
- `knowledge_signals`
- `communication_style`
- `work_preferences`
- `blind_spots`
- `highlight_tags`
- `behavior_vectors`
- `interaction_insights`
- `contrast_signals`
- `confidence`
- `disclaimer`

`evidence` is now a structured array. Each item contains:

- `domains`
- `behavior`
- `strength`
- `is_cross_domain`
- `is_distinctive`

`interaction_insights` is a structured object rather than free text. It contains:

- `relationship_summary`
- `core_collaborators`
- `frequent_people`
- `frequent_chats`

Each list item contains `display_name`, `summary`, and `evidence`.
The final report intentionally strips internal identifiers such as `open_id` and `chat_id`, and `frequent_chats` is constrained to group-chat signals rather than P2P analysis.

The collector now computes a compact pre-LLM chat summary from user-authored chat anchors plus bounded history enrichment. Chat collection resolves the current session user's open_id from the session-local `lark-cli` config, searches only that user's messages, filters obvious non-work groups with a local keyword blacklist, fetches retained chat histories through explicit `+chat-messages-list` page-token pagination, and then extracts compact causal-chain context around the user's own speech. P2P keeps the latest non-noisy counterpart trigger block before each user reply, while groups keep one recent non-noisy topic anchor and add inline role markers such as `[群发起话题]`, `[回应他人]`, and `[被@后回复]`. The same cleaned context feeds both prompt construction and interaction summary ranking.

Document collection uses two inputs before body fetch:

1. current-user-created docs filtered through `creator_ids=[current user open_id]`, capped at 10
2. recently opened docs, capped at 20

The two sets are merged and deduplicated for `docs_summary_reference`, while bounded `docs +fetch` now targets the first 10 current-user-created documents. The final prompt keeps raw `docs` search results plus raw `docs_content` for those 10 documents, and document-body evidence is weighted ahead of chat when inferring stable work patterns and knowledge signals.

`behavior_vectors` must always be a fixed 6-item array in this exact order:

1. `协作方式`: `独立成局` ↔ `高频协同`
2. `表达风格`: `克制压缩` ↔ `高频输出`
3. `决策路径`: `证据校准` ↔ `直觉快判`
4. `推进节奏`: `稳态推进` ↔ `高压突进`
5. `信息处理`: `深度聚焦` ↔ `广度扫描`
6. `风险态度`: `防御优先` ↔ `进攻优先`

The prompt now uses a staged protocol:

- raw domain outputs now keep prompt-time structure rather than digest prose: `chat` becomes a cleaned raw JSON payload with lightweight chat stats, while `docs`, `docs_content`, and `task` enter the prompt as raw JSON
- the chat cleaner removes low-value confirmations, greetings, emojis, repeated short fillers, stickers, image placeholders, and standalone links, while preserving task, timing, decision, risk, and coordination messages together with trigger/topic context
- `mail` is filtered first to remove obvious system/notification traffic, then the filtered `mail` / `mail_content` raw JSON enters the prompt
- `calendar`, `vc` / `vc_notes`, and `docs_content` retain their top-level JSON wrapper in the final prompt, but prompt-time field slimming removes IDs, jump links, log IDs, version notices, avatars, and similar debugging metadata; raw session logs remain unchanged

1. extract behavior facts by domain
2. compare cross-domain consistency and contradictions
3. choose the best BSPI persona and explain the top-2 disambiguation

For the first BOE deployment, the service stays as a Gin HTTP application rather than moving to Kitex. SCM builds a Linux binary and packages it with `photos/`, `web/dist/`, and a repo-owned `bootstrap.sh`. TCE is expected to inject the required `LLM_*` variables plus optional Feishu secrets such as `LARK_APP_ID` and `LARK_APP_SECRET` through runtime environment variables, while `bootstrap.sh` defaults the bind address to `0.0.0.0:8787` and the writable data directory to the packaged `data/` path.

The HTTP service continues to keep `GET /api/sessions/{id}/report` as a compatibility HTML fallback, while the React frontend consumes the new `GET /api/sessions/{id}/report-data` endpoint plus additive status fields `progress`, `events`, `next_action`, and `app_config_required`.

The status endpoint does not add a new failure-detail field. Instead, the frontend renders different retry affordances from the explicit `status` value:

- `auth_failed` continues the authorization flow
- `analysis_failed` offers direct analysis retry without forcing a new authorization
- legacy `failed` remains readable and falls back to generic authorization-style failure presentation

The deployed service exposes `GET /healthz` as a process-level health endpoint. It is intentionally shallow and only confirms that the HTTP server is up, which keeps it safe for liveness and readiness checks during the first single-instance BOE rollout.
