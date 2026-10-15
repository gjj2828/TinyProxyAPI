# 补全遗漏的协议端点设计

## 概述

为 TinyProxyAPI 补全两个此前遗漏的透传端点：

- `POST /v1/messages/count_tokens` —— Anthropic Messages API 估算消息 input token 数（Claude Code CLI `/context` 指令会调用）
- `POST /v1/responses/input_tokens` —— OpenAI Responses API 估算响应 input token 数（Codex CLI 长上下文预检会调用）

两个端点都携带 `model` 字段，与现有 `/v1/messages`、`/v1/responses` 行为完全对称：按 (协议, 模型) 路由到上游同名端点，原样透传 body，替换 Host 与鉴权头。

## 设计原则

- 透传优先：不读 body 内容、不解析参数、不做语义转换。
- 改动最小：复用现有 ServeHTTP 的精确路径匹配 → 协议映射 → body 解析 → matcher 路由 → ReverseProxy 透传全链路。
- 对称一致：count_tokens 走 anthropic 协议（自动用 `x-api-key` + 默认 `anthropic-version`），input_tokens 走 responses 协议（自动用 `Authorization: Bearer`），与各自母端点行为完全一致。

## 接口规范

两个端点的对外契约：

| 项目 | count_tokens | input_tokens |
|------|--------------|----------------|
| Method | `POST` | `POST` |
| Path | `/v1/messages/count_tokens` | `/v1/responses/input_tokens` |
| 鉴权 | `Authorization: Bearer` 或 `x-api-key`（白名单同 `allowed_keys`） | 同左 |
| Body | Anthropic Messages API 的 messages/messages 数组 + model | OpenAI Responses API 的 input + model |
| 上游 | `POST <platform.base_url>/v1/messages/count_tokens` | `POST <platform.base_url>/v1/responses/input_tokens` |

### 响应

原样透传上游响应，代理层不做任何加工。

### 错误响应

完全沿用现有 Error handling，**不新增任何错误类型**：

- 未注册端点 → 404
- body 不是合法 JSON 或缺 `model` → 400 `invalid_request_error`
- 客户端鉴权缺失或不匹配 → 401 `invalid_request_error`
- 模型未配置且无 default platform → 404 `invalid_request_error`（message 给出配置提示）
- 上游返回的错误 → 原样透传

## 实现

### 端点协议表扩展

修改 `handler/proxy.go` 第 30-34 行 `endpointProtocol` map，从 3 项扩展到 5 项：

```go
var endpointProtocol = map[string]string{
    "/v1/responses":              protocolResponses,
    "/v1/responses/input_tokens": protocolResponses, // 新增
    "/v1/chat/completions":       protocolOpenAI,
    "/v1/messages":               protocolAnthropic,
    "/v1/messages/count_tokens":  protocolAnthropic, // 新增
}
```

注释从「支持的协议种类」改为更精确的描述，明确指出每个端点的协议归属。

### 文件变更

- `handler/proxy.go`：扩展 `endpointProtocol` map + 更新注释
- `handler/proxy_test.go`：在 `TestProxyHandler_ProtocolRouting` 的 `cases` 切片中追加 2 个 case
- `README.md`：更新端点列表（已支持三种 → 五种）+ 协议说明段落追加 2 行 curl 示例

### 代码逻辑

```
ServeHTTP:
  if r.Method == "GET" && r.URL.Path == "/v1/models":
    → 现有 handleListModels 逻辑不变
  else if r.Method == "POST" && r.URL.Path in endpointProtocol:
    → 现有逻辑不变（自动覆盖 count_tokens、input_tokens）
  else:
    → 404 Not Found
```

`protocol` 变量由 `endpointProtocol[r.URL.Path]` 查得，自动决定鉴权头：`protocolAnthropic` → `x-api-key`，否则 → `Authorization: Bearer`。两个新增端点自动获得正确的鉴权头处理，无须特判。

### 热重载

无影响——端点协议表是常量，重启后生效；配置热重载不影响端点集合。

## 测试要点

在 `handler/proxy_test.go` 的 `TestProxyHandler_ProtocolRouting` 的 `cases` 切片里追加：

- `name: "anthropic_count_tokens"`：路径 `/v1/messages/count_tokens`，`apiKeyHdr: "client-key"`，断言上游路径相同、上游 `x-api-key=sk-upstream`、上游 `Authorization` 为空、上游 `anthropic-version` 已设置。
- `name: "responses_input_tokens"`：路径 `/v1/responses/input_tokens`，`authHdr: "Bearer client-key"`，断言上游路径相同、上游 `Authorization=Bearer sk-upstream`、上游 `x-api-key` 为空。

复用现有 `capturedCh` mock upstream 即可（按路径分支即可同时捕获 5 个端点）。

## 不补的端点（明确范围外）

| 路径 | 原因 |
|------|------|
| `/v1/messages/batches` 系列 | 跨平台一致性问题：批量结果只能从提交时同一平台读取，记忆方案不可靠 |
| `/v1/completions`、`/v1/embeddings`、`/v1/images/*` | 与 model 路由语义一致性低，且使用频率低 |
| `/v1/batches` 系列 | 与 `/v1/messages/batches` 同问题 |
| `/v1/responses/{response_id}`（GET/cancel/input_items） | GET/cancel 请求体不携带 model，无法按模型路由；默认方案又隐含 text 协议平台限定 |
| `/v1/audio/*`、`/v1/files`、`/v1/fine_tuning/*` 等管理类端点 | 与「按 model 路由」无关 |

## 兼容性

- **向后兼容**：现有 3 个端点行为不变；新增端点仅当客户端显式调用才生效。
- **配置文件**：无须任何配置变更，端点协议表是代码常量。
- **客户端**：Codex CLI 长上下文预检、Claude Code CLI `/context` 指令调用将在升级后自动走通。