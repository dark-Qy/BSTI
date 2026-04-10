# AGENTS.md

## Required Reading Before Code Changes

在修改代码前，先阅读：

1. `AGENTS.md`
2. `ARCHITECTURE.md`
3. `README.md`
4. `docs/design-docs/` 下相关专题文档
5. `docs/product-specs/` 下相关能力说明
6. `docs/exec-plans/` 下相关执行计划
7. `rules/` 下协作与验证规则

## Engineering Role

默认设计角色是一名高级工程师。

这意味着：

- 优先按长期可维护性设计，而不是只追求功能堆叠
- 复杂后端逻辑必须先划清职责边界，再继续扩展
- 当文件过长、职责混杂、review 成本明显上升时，应优先做结构拆分

## Documentation Governance

当修改以下任一内容时，必须同步更新文档：

- message envelope 协议
- action 协议
- world state 结构
- memory view 结构
- HTTP API
- agent manifest 规范

至少同步更新：

- `README.md`
- `ARCHITECTURE.md`
- `docs/design-docs/` 下对应专题文档
- `docs/product-specs/` 下对应产品说明

## Working Rule

- 长期设计放在 `docs/design-docs/`
- 用户视角能力说明放在 `docs/product-specs/`
- 执行状态只放在 `docs/exec-plans/`
- 规则文档只放在 `rules/`
- 复杂后端模块必须按职责分层；
- 新增或重构后端能力前，先检查 `rules/backend-design-principles.md`

## Minimal Vocabulary

- `Agent`: 通过 HTTP 接入的平台外部智能体
- `MessageEnvelope`: 平台统一消息信封
- `ActionGateway`: 校验和应用状态变更的网关
- `WorldState`: 平台共享环境状态快照
- `MemoryView`: 平台派生的 Agent 记忆视图

## Validation Index

- 提交前验证见 `rules/validation-gate.md`
- 接口兼容性规则见 `rules/agent-interface-compatibility.md`
- 文档更新策略见 `rules/design-doc-update-policy.md`
- 后端设计规范见 `rules/backend-design-principles.md`
