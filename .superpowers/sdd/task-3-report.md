# Task 3 Report: ProxyHandler 添加 API Key 验证

## 概述

在 `ProxyHandler.ServeHTTP` 中添加 API Key 白名单验证逻辑。

## 修改内容

**文件:** `handler/proxy.go`

在平台匹配之后、ReverseProxy 之前添加了 API Key 验证：

1. 读取 `Authorization` 请求头
2. 若缺失，返回 401 `missing authorization header`
3. 提取 `Bearer <key>` 中的 key，检查是否在 `h.matcher.GetAllowedKeys()` 白名单中
4. 若不匹配，返回 401 `invalid api key`

## 提交信息

- **Commit SHA:** 88f9c3eb41d9ac7b2946c7b7929620295357bd58
- **Subject:** feat: add API key whitelist validation in ProxyHandler

## 测试结果

```
=== RUN   TestProxyHandler_ServeHTTP
--- PASS: TestProxyHandler_ServeHTTP (0.00s)
PASS
ok  	github.com/user/TinyProxyAPI/handler	1.189s
```

所有测试通过。

## 验收标准

- [x] `handler/proxy.go` 中 ServeHTTP 方法包含 API Key 验证逻辑
- [x] 缺失 Authorization header 返回 401
- [x] 无效 API Key 返回 401
- [x] 测试通过
- [x] 提交已创建
