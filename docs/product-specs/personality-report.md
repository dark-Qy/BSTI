# Personality Report Product Spec

Users open the local web page, configure a Feishu app if the service has not been preconfigured, authorize Feishu access, and request analysis. Session-local app setup is never reused across sessions; only service-side `LARK_APP_ID` and `LARK_APP_SECRET` can skip the first step. The product flow is divided into three UI stages:

- Landing & Auth
- Analysis & Loading
- Result & Persona Report

The frontend should preserve the current local `session_id` in browser `localStorage` so refreshing or reopening the same browser can continue the authorized flow without reauthorizing. This stored value is only a session reference; the real Feishu login state remains inside the server-side session directory. Different browsers and browser profiles are isolated naturally, while a shared browser must expose an explicit "切换账号 / 新建会话" entry so a second user can discard the previous local session reference.

The result page renders from structured JSON and presents:

- a single BSPI Top1 persona with image
- 2-4 highlight tags
- six fixed behavior vectors
- a `工作画像` module
- an `表达指纹` module
- a `知识信号` module
- an interaction insights module with relationship summary, core collaborators, frequent people, and frequent chats
- contrast signals when the model finds cross-domain contradictions or notable behavior contrast
- official persona definition
- evidence-based individual summary
- evidence-based observations
- communication style
- work preference notes
- likely blind spots
- confidence and data coverage notes
- local poster export action
- disclaimer that this is not a diagnosis

The report generation backend is configured through `LLM_PROVIDER`, `LLM_API_URL`, `LLM_API_KEY`, `LLM_MODEL`, and `LLM_MAX_TOKENS`. Users explicitly choose `modelhub` or `kimi`; the product does not auto-detect provider type from the URL and allows custom compatible gateway URLs.

Authorization now requests one-time read-only chat coverage broad enough for:

- cross-chat message search
- direct-message history reads
- group chat history reads
- chat metadata lookup

Chat-derived interaction insights should be built from the user's own messages rather than all visible traffic. The product should:

- search for messages authored by the authenticated user
- filter obvious non-work groups with local heuristics before ranking
- extract compact causal-chain context around the user's own messages by paginating `+chat-messages-list` with returned page tokens
- for P2P, keep the most recent non-noisy counterpart trigger message block before each user reply
- for groups, keep one recent non-noisy topic anchor and inline role markers such as `[群发起话题]`, `[回应他人]`, and `[被@后回复]`
- prioritize P2P when identifying core collaborators, while keeping groups as supplemental coordination evidence
- hide internal identifiers such as `open_id` and `chat_id` from final JSON/HTML/Markdown report outputs
- keep `frequent_chats` limited to group-chat findings rather than single-chat summaries or direct-message evidence

Document collection now prioritizes:

1. current-user-created docs, capped at 10 and filtered with `creator_ids=[current user open_id]`
2. recently browsed docs, capped at 20

The two inputs are merged and deduplicated for reference display, while bounded body reads now target the first 10 current-user-created documents. Before the final analysis call, `chat` is cleaned into a filtered raw JSON block plus lightweight chat stats, `docs` / `docs_content` / `task` are passed through as raw JSON, and `calendar`, `vc` / `vc_notes`, and filtered `mail` / `mail_content` also remain JSON blocks with their original top-level wrapper but slimmer field sets. Prompt-time slimming removes internal identifiers, jump links, avatars, CLI update notices, log IDs, thread IDs, and similar noise while keeping the human-readable content unchanged. Raw session logs still preserve the original CLI responses. When both chat and docs are available, docs should contribute more strongly to stable work-pattern, writing-style, and knowledge-signal analysis.

The six behavior vectors are fixed and must stay stable across reports:

| Label | Left Pole | Right Pole |
| --- | --- | --- |
| 协作方式 | 独立成局 | 高频协同 |
| 表达风格 | 克制压缩 | 高频输出 |
| 决策路径 | 证据校准 | 直觉快判 |
| 推进节奏 | 稳态推进 | 高压突进 |
| 信息处理 | 深度聚焦 | 广度扫描 |
| 风险态度 | 防御优先 | 进攻优先 |

For deployed environments, the same HTTP service also exposes `GET /healthz` with a lightweight `{"status":"ok"}` response so BOE/TCE can confirm the process is ready before routing traffic.

The frontend also relies on:

- `GET /api/sessions/{id}/status` additive fields `progress`, `events`, `next_action`, and `app_config_required` so the landing page can distinguish manual two-step auth from service-preconfigured auth
- `GET /api/sessions/{id}/report-data` structured report JSON

Frontend restore behavior should be:

- restore the persisted `session_id` first when present
- auto-create a new session only if that restore returns `404`
- keep non-404 restore failures visible instead of silently resetting the flow

Failure handling should follow explicit backend status values instead of inferring cause from the current UI step:

- `auth_failed`: show an authorization failure state and guide the user back into setup or Feishu authorization
- `analysis_failed`: show an analysis failure state that explains authorization already succeeded and offers direct retry without reauthorizing
- legacy `failed`: continue to render as a generic authorization-style failure for backward compatibility

The `report-data` payload now also carries these portrait blocks:

- `work_profile`
- `expression_fingerprint`
- `output_style`
- `knowledge_signals`
- `contrast_signals`

## Persona Shorthand

The report and its supporting visual assets should use memorable persona shorthand that favors common English words over compressed pseudo-acronyms. Each persona keeps a stable shorthand plus the existing Chinese label.

| Shorthand | Chinese Label |
| --- | --- |
| BLAZE | 纵火者 |
| AGILE | 狂奔者 |
| KEEN | 雷达怪 |
| PRISM | 变色龙 |
| BOND | 人肉担保 |
| FRANK | 嘴替 |
| CLEAR | 人形提炼机 |
| CALM | 灭火器 |
| CORE | 刨坟者 |
| DATA | 数据教徒 |
| REAL | 验尸官 |
| DARE | 赌徒 |
| APEX | 偏执狂 |
| SCOUT | 框架逃逸者 |
| ROI | 人形计算器 |
| GRIT | 打不死的 |
| GROW | 永远在学的 |
| NORTH | 布道者 |
| SPARK | 脑洞制造机 |
| HUMBLE | 自我审计员 |
