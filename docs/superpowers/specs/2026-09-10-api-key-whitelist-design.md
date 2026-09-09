# API Key 白名单验证设计文档

## 概述

为 TinyProxyAPI 添加全局 API Key 白名单验证功能，确保只有持有合法 key 的客户端才能发起请求。

## 背景

当前 TinyProxyAPI 对客户端请求没有任何认证，任何知道服务地址的客户端都可以发起请求。添加白名单验证可以：
- 防止未授权的请求
- 控制哪些客户端可以使用代理服务

## 功能范围

- 新增 `[allowed_keys]` 配置节，支持多个 key
- 验证客户端请求中 `Authorization: Bearer <key>` 是否在白名单中
- 不匹配时返回 401

## 配置变更

### config.toml

```toml
[server]
addr = ":3178"

[default]
platform = "openai"

[allowed_keys]
keys = ["sk-client-key-1", "sk-client-key-2"]

[platforms]
"openai" = { base_url = "https://api.openai.com", api_key = "sk-xxx" }

[models]
"gpt-4o" = "openai"
```

## Config 结构变更

### config/loader.go

```go
type Config struct {
    Server     ServerConfig
    Default    DefaultConfig
    AllowedKeys []string         // 新增：全局 API Key 白名单
    Platforms  map[string]Platform
    Models     map[string]string
}
```

## 实现

### handler/proxy.go

在 `ServeHTTP` 中，解析 model 之后、转发请求之前，插入验证逻辑：

```go
// 提取 Authorization header 中的 Bearer token
authHeader := r.Header.Get("Authorization")
if authHeader == "" {
    w.WriteHeader(http.StatusUnauthorized)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "error": map[string]string{
            "message": "missing authorization header",
            "type":    "invalid_request_error",
        },
    })
    return
}

// 验证 Bearer token 是否在白名单中
isValid := false
for _, key := range h.matcher.GetAllowedKeys() {
    if authHeader == "Bearer "+key {
        isValid = true
        break
    }
}
if !isValid {
    w.WriteHeader(http.StatusUnauthorized)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "error": map[string]string{
            "message": "invalid api key",
            "type":    "invalid_request_error",
        },
    })
    return
}
```

### matcher/matcher.go

新增 `GetAllowedKeys()` 方法：

```go
func (m *Matcher) GetAllowedKeys() []string {
    return m.cfg.AllowedKeys
}
```

## 错误处理

| 场景 | 响应状态码 | 响应体 |
|------|-----------|--------|
| 缺少 Authorization header | 401 | `{"error": {"message": "missing authorization header", "type": "invalid_request_error"}}` |
| API key 不在白名单中 | 401 | `{"error": {"message": "invalid api key", "type": "invalid_request_error"}}` |

## 测试策略

- 单元测试：ConfigLoader 解析 allowed_keys
- 单元测试：Matcher.GetAllowedKeys()
- 集成测试：ProxyHandler 验证逻辑（401 场景）
