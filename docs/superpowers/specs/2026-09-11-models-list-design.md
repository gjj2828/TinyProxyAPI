# Models List 接口设计

## 概述

新增 `GET /v1/models` 接口，以 OpenAI 兼容格式返回当前配置的模型列表，供下游客户端发现可用模型。

## 接口规范

| 项目 | 值 |
|------|-----|
| Method | `GET` |
| Path | `/v1/models` |
| 鉴权 | `Authorization: Bearer <key>`（与 `/v1/responses` 共用 `allowed_keys` 白名单） |
| 响应 Content-Type | `application/json` |

### 成功响应 (200 OK)

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-4o",
      "object": "model",
      "created": 1788944949,
      "owned_by": "openai"
    },
    {
      "id": "gpt-4o-mini",
      "object": "model",
      "created": 1788944949,
      "owned_by": "openai"
    }
  ]
}
```

字段说明：
- `id`：模型名，来源于配置 `[models]` 表的 key
- `object`：固定为 `"model"`
- `created`：固定为 `1788944949`（2026-09-09 09:09:09 UTC），后续可扩展为配置驱动
- `owned_by`：模型所属平台名，来源于配置 `[models]` 表的 value（即 `[platforms]` 中的 key）

### 错误响应

**未提供 Authorization header (401)：**
```json
{
  "error": {
    "message": "missing authorization header",
    "type": "invalid_request_error"
  }
}
```

**API Key 不在白名单中 (401)：**
```json
{
  "error": {
    "message": "invalid api key",
    "type": "invalid_request_error"
  }
}
```

## 实现

### 文件变更

- `handler/proxy.go`：`ServeHTTP` 方法新增 `GET /v1/models` 分支

### 代码逻辑

```
ServeHTTP:
  if r.Method == "GET" && r.URL.Path == "/v1/models":
    → 鉴权（复用现有 validateAuth() 逻辑）
    → 从 currentMatcher().cfg.Models 遍历生成 model 列表
    → 返回 200 JSON
  else if r.Method == "POST" && r.URL.Path == "/v1/responses":
    → 现有逻辑不变
  else:
    → 404 Not Found
```

### 热重载

每次请求通过 `currentMatcher()` 获取最新 matcher，配置热重载后自动反映新模型列表。

## 测试要点

- [ ] 未带 Authorization header 返回 401
- [ ] 带无效 API Key 返回 401
- [ ] 带有效 API Key 返回 200 和模型列表
- [ ] 模型数量与 `[models]` 配置项数量一致
- [ ] 每个模型的 `id`、`owned_by` 与配置一致
- [ ] 热重载后列表内容更新
