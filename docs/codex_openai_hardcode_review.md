# Codex/OpenAI Hardcode Review

本文档只做代码定位和最小改动评估，不包含业务代码修改。

## 问题 1：模型仍有多处硬编码，新增自定义模型不能只改一个地方

### 当前位置

- `backend/internal/pkg/openai/constants.go:17`
  - `DefaultModels` 写死了 OpenAI/Codex 模型列表。
- `backend/internal/pkg/openai/constants.go:40`
  - `DefaultTestModel` 写死为 `gpt-5.1-codex`。
- `backend/internal/service/openai_codex_transform.go:11`
  - `codexModelMap` 写死了模型别名和归一化映射。
- `backend/internal/service/openai_codex_transform.go:147`
  - `normalizeCodexModel` 继续把很多模型名硬编码归一化。
- `backend/internal/handler/gateway_handler.go:815`
  - `/v1/models` 在 fallback 场景直接返回 `openai.DefaultModels`。
- `backend/internal/handler/admin/account_handler.go:1537`
- `backend/internal/handler/admin/account_handler.go:1544`
  - 管理端账号可用模型列表 fallback 也直接返回 `openai.DefaultModels`。
- `frontend/src/components/account/BulkEditAccountModal.vue:941`
  - `allModels` 为本地硬编码列表。
- `frontend/src/components/account/BulkEditAccountModal.vue:972`
  - `presetMappings` 为本地硬编码映射。
- `frontend/src/composables/useModelWhitelist.ts:232`
  - `allModelsList` 为本地硬编码模型全集。
- `frontend/src/composables/useModelWhitelist.ts:272`
  - OpenAI 预设映射仍为硬编码。
- `frontend/src/components/keys/UseKeyModal.vue:616`
  - Key 使用引导里的模型说明和默认模型映射仍为硬编码。

### 已有能力

- `backend/internal/service/account.go:503`
  - `IsModelSupported` 已支持按 `model_mapping` 做精确匹配和通配符匹配。
- `backend/internal/service/account.go:523`
  - `GetMappedModel` 已支持把请求模型映射到任意上游模型。
- `backend/internal/service/gateway_service_antigravity_whitelist_test.go`
  - 测试已经证明后端核心路由允许不在默认列表里的自定义模型通过。

### 结论

后端核心调度本身并没有完全卡死在默认模型列表上，真正写死的是以下三层：

- 展示层模型列表
- Codex 模型归一化表
- 默认测试模型和若干前端预设

这意味着“支持自定义新模型”不是重构级工作，主要是把硬编码注册表从多个文件收敛到一个可配置来源。

### 最小改动建议

- 后端先引入一个统一的 OpenAI/Codex 模型注册表。
  - 可来自配置、数据库设置或单独 JSON 文件。
- `DefaultModels`、`codexModelMap`、`DefaultTestModel` 改为从统一注册表读取，默认值仍可保留。
- `/v1/models` 和 `/api/v1/admin/accounts/:id/models` 统一从注册表返回。
- 前端不要再维护本地 OpenAI 模型全集。
  - 最小方案是读取后端接口返回值。
  - 次小方案是至少把前端列表改成从单一配置文件生成。

### 最小改动成本评估

- 后端路由与调度：低
- 后端模型归一化：中
- 前端 UI 联动：中

## 问题 2：MD 提示词目前不是只跟随 Codex 请求头发送，非 Codex agent 也会带上

### 当前位置

- `backend/internal/service/openai_gateway_service.go:1458`
  - `isCodexCLI` 只用 `User-Agent` 的 `IsCodexCLIRequest(...)` 判断，或 `ForceCodexCLI` 兜底。
- `backend/internal/pkg/openai/request.go:14`
- `backend/internal/pkg/openai/request.go:28`
- `backend/internal/pkg/openai/request.go:44`
- `backend/internal/pkg/openai/request.go:53`
  - 仓库里其实已经有更宽的“官方 Codex 客户端”判定。
  - 包括 `User-Agent` 和 `originator` 两套检测，但主链路没有统一使用。
- `backend/internal/service/openai_gateway_service.go:1567`
  - 非 Codex CLI 且 `instructions` 为空时，仍会注入 `GetOpenCodeInstructions()`。
- `backend/internal/service/openai_codex_transform.go:232`
  - `GetOpenCodeInstructions()` 实际返回的仍是内嵌提示词。
- `backend/internal/service/openai_codex_transform.go:244`
  - `applyInstructions` 按 `isCodexCLI` 分支处理。
- `backend/internal/service/openai_codex_transform.go:253`
  - `applyCodexCLIInstructions` 会给 Codex CLI 请求补提示词。
- `backend/internal/service/openai_codex_transform.go:269`
  - `applyOpenCodeInstructions` 会给非 Codex 请求写入同一套内嵌提示词。
- `backend/internal/service/openai_gateway_service.go:2624`
- `backend/internal/service/openai_gateway_service.go:2626`
  - HTTP 上游请求对 OAuth 账号会强制设置 `originator`。
  - Codex 是 `codex_cli_rs`，非 Codex 是 `opencode`。
- `backend/internal/service/openai_ws_forwarder.go:1145`
- `backend/internal/service/openai_ws_forwarder.go:1147`
  - WebSocket 链路也做了同样的 `originator` 注入。
- `backend/internal/service/openai_gateway_service.go:3772`
  - Codex 模型的 passthrough 场景如果没有 `instructions` 会被直接拒绝。
- `backend/internal/service/prompts/codex_cli_instructions.md:1`
  - 主链路 MD 提示词文件。
- `backend/internal/pkg/openai/instructions.txt:1`
  - 另一份编译时内嵌提示词，目前主要在测试账号链路使用。

### 问题表现

- 当前实现并不是“只有 Codex 头才发送 MD”。
- 对 OpenAI OAuth 链路来说，非 Codex 请求也会被补 `instructions`。
- 非 Codex 请求还会被额外打上 `originator=opencode`。
- `isCodexCLI` 判定又比仓库里现成的“官方 Codex 客户端”规则更窄。

### 结论

如果你的目标是：

- 只有 Codex 请求头命中时才发送 MD
- 其余 agent 不发送 MD

那么当前实现不满足。

### 最小改动建议

- 增加一个统一判断函数，例如 `shouldAttachCodexInstructions(...)`。
- 判断条件不要只看 `IsCodexCLIRequest(User-Agent)`。
  - 应至少统一到现有的官方客户端检测：
  - `IsCodexOfficialClientRequest(User-Agent)`
  - `IsCodexOfficialClientOriginator(originator)`
  - `ForceCodexCLI`
- 只有命中这个统一判断时才执行：
  - `applyCodexCLIInstructions`
  - `originator=codex_cli_rs`
- 非 Codex 请求：
  - 不注入 MD 提示词
  - 不再主动发 `originator=opencode`

### 最小改动成本评估

- `instructions` 注入收口：低
- HTTP/WS 两条链路统一判断：低
- 与现有 passthrough 拒绝逻辑对齐：中

## 问题 3：推理强度不能只依赖请求体，应能直接跟随模型后缀同步

### 当前位置

- `backend/internal/service/openai_gateway_service.go:1600`
  - 目前只对 `reasoning.effort=minimal` 做了归一化。
- `backend/internal/service/openai_gateway_service.go:3684`
  - `getOpenAIReasoningEffortFromReqBody` 先从请求体读取。
- `backend/internal/service/openai_gateway_service.go:3704`
  - `deriveOpenAIReasoningEffortFromModel` 才从模型后缀推导。
- `backend/internal/service/openai_gateway_service.go:3791`
  - `extractOpenAIReasoningEffortFromBody` 的优先级也是请求体优先，模型后缀兜底。
- `backend/internal/service/openai_gateway_service.go:3830`
  - `extractOpenAIReasoningEffort` 也是同样逻辑。
- `backend/internal/service/openai_gateway_service.go:3845`
  - `normalizeOpenAIReasoningEffort` 只允许 `low/medium/high/xhigh`。
- `backend/internal/service/openai_gateway_service_hotpath_test.go`
  - 现有测试明确验证了“请求体优先、模型后缀兜底”的行为。

### 问题表现

- 当前逻辑更像“记录推理强度”，不是“同步推理强度”。
- 代码会从模型名里推导 `low/high/xhigh`，但主要用于记录和统计。
- 它不会在主链路里把最终模型后缀稳定写回到 outgoing request body。
- 如果模型选择了 `...-low`、`...-high`、`...-xhigh`，而请求体没带或带了冲突值，最终上游请求不一定同步。

### 结论

这和你要的行为不同。你要的是：

- 推理强度可以直接跟模型选择同步
- 不能永远以请求体的 `reason` 或 `reasoning.effort` 为准

当前实现只满足“读取”与“记账”，不满足“同步写回”。

### 最小改动建议

- 在模型归一化完成后，统一得到最终 `effectiveModel`。
- 从 `effectiveModel` 推导最终 `effectiveReasoningEffort`。
- 当请求体没有 `reasoning.effort` 时，自动补上模型后缀对应值。
- 如果产品规则是“模型优先于请求体”，那就直接覆盖请求体里的冲突值。
- 如果产品规则是“显式请求体优先”，那至少要在模型后缀和请求体不一致时提供统一策略，不要只做记录。

### 最小改动成本评估

- 只做“缺失时自动补齐”：低
- 做“模型优先覆盖请求体”：中
- HTTP/WS/透传三链路完全一致：中

## 附录：`Content-Type` 相关地址

### 当前位置

- `frontend/src/api/client.ts:18`
- `frontend/src/api/client.ts:187`
  - 前端 axios 默认发送的是精确值 `application/json`。
- `backend/internal/service/openai_gateway_service.go:55`
- `backend/internal/service/openai_gateway_service.go:68`
  - `content-type` 被加入了普通模式和 passthrough 模式的请求头白名单。
- `backend/internal/service/openai_gateway_service.go:2191`
  - `buildUpstreamRequestOpenAIPassthrough(...)` 会先复制入站请求头。
- `backend/internal/service/openai_gateway_service.go:2277`
  - passthrough 只在缺失时补 `application/json`。
- `backend/internal/service/openai_gateway_service.go:2570`
  - `buildUpstreamRequest(...)` 普通模式同样先复制允许的请求头。
- `backend/internal/service/openai_gateway_service.go:2648`
  - 普通模式也只是“缺失时补 `application/json`”。

### 结论

主链路里我没有找到“显式拒绝 `application/json; charset=utf-8`”的校验。

当前真实风险不是“本地校验不通过”，而是：

- 如果客户端传进来的是 `application/json; charset=utf-8`
- 网关会把它原样透传到上游
- 因为现有代码只在 header 缺失时才补成精确 `application/json`

### 最小改动建议

- 如果上游必须严格要求精确值 `application/json`，最小改动点就是：
  - `buildUpstreamRequestOpenAIPassthrough(...)`
  - `buildUpstreamRequest(...)`
- 在这两个函数里不要“缺失才补”，而是统一覆盖成精确值 `application/json`。

## 总结

从当前代码看，这三个需求都不是重构级别：

- 自定义新模型：核心路由已有基础能力，主要是收敛硬编码注册表
- MD 只随 Codex 请求头发送：主要是统一识别条件和关掉非 Codex 注入
- 推理强度跟模型同步：主要是把“推导”升级成“写回策略”

真正适合先动手的最小改动入口，优先级建议如下：

1. `backend/internal/service/openai_gateway_service.go`
2. `backend/internal/service/openai_codex_transform.go`
3. `backend/internal/pkg/openai/constants.go`
4. `frontend/src/composables/useModelWhitelist.ts`
5. `frontend/src/components/account/BulkEditAccountModal.vue`
