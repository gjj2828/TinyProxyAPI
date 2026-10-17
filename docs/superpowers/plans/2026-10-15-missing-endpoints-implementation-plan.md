# 补全遗漏端点（count_tokens / input_tokens）实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-driven-development (recommended) or executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 TinyProxyAPI 透传 `POST /v1/messages/count_tokens` 与 `POST /v1/responses/input_tokens` 两个 token 计数端点，行为与各自母端点完全对称。

**Architecture:** 只扩展 `handler/proxy.go` 的常量 `endpointProtocol` 映射表（3 → 5 项）。`ServeHTTP` 的「精确路径匹配 → 协议查表 → 解析 body 取 model → matcher 路由 → `httputil.ReverseProxy` 透传」全链路一行不改；协议归属自动决定上游鉴权头（anthropic → `x-api-key` + 缺失时补 `anthropic-version`，其余 → `Authorization: Bearer`）。

**Tech Stack:** Go 1.21+、`net/http`、`net/http/httputil`、`encoding/json`、`testing`

**Spec:** `docs/superpowers/specs/2026-10-15-missing-endpoints-design.md`（随本计划一同阅读）

## Global Constraints

- 路径**精确匹配**：`endpointProtocol` 是 `map[string]string` 精确查表，禁止改成前缀匹配、正则匹配或通配。
- **不新增任何错误类型**：完全沿用现有 Error handling —— 未注册端点 404、body 非合法 JSON 400、鉴权缺失/不匹配 401、模型未配置且无 default platform 404、上游错误原样透传。
- **不解析 body 语义**：body 只 `json.Unmarshal` 出 `model` 用于路由，随后 `r.Body` 用同一份字节完整还原并原样转发；不校验 messages / input / tools 等其它字段。
- **不新增配置项**：端点集合是代码常量，`config.toml` 格式零变化，配置热重载无影响。
- **凭证不泄露**：Director 先 `Del` 客户端的 `Authorization` 与 `x-api-key`，再写平台 key；客户端已带 `anthropic-version` 时**不得**覆盖，仅在缺失时补默认值 `2023-06-01`。
- **向后兼容**：现有三个端点 `/v1/responses`、`/v1/chat/completions`、`/v1/messages` 的行为逐条不变。
- **明确不做**（spec 第 95-103 行范围外）：`/v1/messages/batches`、`/v1/batches`、`/v1/completions`、`/v1/embeddings`、`/v1/images/*`、`/v1/responses/{response_id}`、`/v1/audio/*`、`/v1/files`、`/v1/fine_tuning/*`。
- **本机沙盒下跑 Go 命令前必须重定向缓存**，否则会被 ACL 拒绝：
  ```powershell
  $env:GOCACHE="$env:TEMP\tpa-gocache"; $env:GOMODCACHE="$env:TEMP\tpa-gomodcache"
  ```
  基线（改动前）`go test ./...` 已确认 7 个包全绿，可作为回归对照。
- **已知并接受的 spec 偏差**：spec 第 39 行写「缺 `model` → 400」，但现有 `ServeHTTP` 对 `model` 为空的合法 JSON 会走 `matcher.Match(protocol, "")` → 返回 404 `model not configured`（若配了 `[default].platform` 则透传给 default 平台）。本计划**保持现状、不新增校验**，以免与「现有三个端点行为不变」冲突。Task 1 用测试把实际行为钉死。

## Review Focus

以下五类是 spec 隐含、但 spec 自己的测试要点没覆盖、且最容易在真实使用中咬人的情况。每条都已落到 Task 1 的具体步骤。

- **非 POST 方法**（`GET /v1/messages/count_tokens`、`GET /v1/responses/input_tokens`）：应返回 404，不得因端点进表而被放行 → Task 1 Step 5
- **路径变体**（尾斜杠 `/v1/messages/count_tokens/`、多余路径段 `/v1/responses/input_tokens/extra`）：应返回 404，防止将来有人把精确查表改成前缀匹配 → Task 1 Step 5
- **客户端自带 `anthropic-version`**（Claude Code 一定会带自定义版本）：应原样透传给上游，不被默认值 `2023-06-01` 覆盖 → Task 1 Step 1
- **body 是合法 JSON 但缺 `model` 字段**：应与现有端点一致返回 404 且 `error.message` 含 `model not configured`，且请求**不**发往上游 → Task 1 Step 5
- **客户端同时带 `Authorization: Bearer <白名单key>` 与 `x-api-key: <错误key>`**：白名单判定以 `Authorization` 为准（200），且客户端那个错误 `x-api-key` 绝不能出现在上游请求里 → Task 1 Step 5

---

## 文件结构

| 文件 | 改动 | 职责 |
|------|------|------|
| `handler/proxy.go` | 改：常量注释 + `endpointProtocol` 3 → 5 项 | 端点 → 协议的唯一真相源 |
| `handler/proxy_test.go` | 改：`TestProxyHandler_ProtocolRouting` 的 cases 表与断言分支；新增 `TestProxyHandler_TokenEndpointGuards` | 锁定两个新端点的路由、鉴权头与边界行为 |
| `README.md` | 改：端点清单、协议说明、curl 示例 | 面向用户的能力声明 |
| `README_EN.md` | 改：同上（spec 漏列，本计划补上） | 英文版与中文版保持一致 |

---

### Task 1: 扩展端点协议表

**Files:**
- Modify: `handler/proxy_test.go:189-198`（ProtocolRouting 的 cases 结构体与切片）、`handler/proxy_test.go:200-212`（请求头构造）、`handler/proxy_test.go:222-239`（协议分支断言）
- Modify: `handler/proxy_test.go` 文件末尾（追加 guards 测试）
- Modify: `handler/proxy_test.go:4-13`（import 块）
- Modify: `handler/proxy.go:18`（常量注释）、`handler/proxy.go:30-34`（`endpointProtocol`）

**Interfaces:**
- Consumes:
  - `endpointProtocol map[string]string`（`handler/proxy.go`，key = 服务端精确路径）
  - 常量 `protocolResponses` / `protocolOpenAI` / `protocolAnthropic`，`defaultAnthropicVersion = "2023-06-01"`
  - `config.Config{AllowedKeys []string, Platforms map[string]config.Platform, Models map[string]string}`、`config.Platform{BaseURL, APIKey string}`
  - `matcher.NewMatcher(*config.Config) *matcher.Matcher`、`handler.NewProxyHandler(*matcher.Matcher) *ProxyHandler`、`(*ProxyHandler).ServeHTTP(http.ResponseWriter, *http.Request)`
- Produces: 无新导出符号。端点集合从 3 项扩为 5 项；`TestProxyHandler_TokenEndpointGuards` 为新增测试函数。

- [ ] **Step 1: 写失败测试 —— 扩展 ProtocolRouting 的 cases 表**

编辑 `handler/proxy_test.go` 第 189-198 行，把 cases 结构体与切片替换为：

```go
	cases := []struct {
		name          string
		path          string
		authHdr       string
		apiKeyHdr     string
		wantAnthropic bool   // 是否按 anthropic 协议向上游发凭证
		clientVersion string // 非空表示客户端自带 anthropic-version，期望原样透传
	}{
		{name: "responses", path: "/v1/responses", authHdr: "Bearer client-key"},
		{name: "openai", path: "/v1/chat/completions", authHdr: "Bearer client-key"},
		{name: "anthropic", path: "/v1/messages", apiKeyHdr: "client-key", wantAnthropic: true},
		{name: "anthropic_count_tokens", path: "/v1/messages/count_tokens", apiKeyHdr: "client-key", wantAnthropic: true},
		{name: "responses_input_tokens", path: "/v1/responses/input_tokens", authHdr: "Bearer client-key"},
		{name: "anthropic_count_tokens_client_version", path: "/v1/messages/count_tokens", apiKeyHdr: "client-key", wantAnthropic: true, clientVersion: "2024-10-22"},
	}
```

在同一函数第 209 行 `apiKeyHdr` 的 `if` 块之后追加：

```go
			if tc.clientVersion != "" {
				req.Header.Set("anthropic-version", tc.clientVersion)
			}
```

把第 222 行的 `if tc.name == "anthropic" {` 改为 `if tc.wantAnthropic {`，并把该分支里第 229-231 行的 `anthropic-version` 空值断言替换为：

```go
				if tc.clientVersion != "" {
					if got.anthropicVersion != tc.clientVersion {
						t.Errorf("expected client anthropic-version %q preserved, got %q", tc.clientVersion, got.anthropicVersion)
					}
				} else if got.anthropicVersion == "" {
					t.Errorf("expected default anthropic-version to be set")
				}
```

> 注意：`capturedCh` 缓冲区是 1，子测试串行执行时每个子测试恰好读一条。不要给这些子测试加 `t.Parallel()`。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./handler/ -run TestProxyHandler_ProtocolRouting -v`
Expected: FAIL —— 三个新子测试报 `expected 200, got 404`（`anthropic_count_tokens`、`responses_input_tokens`、`anthropic_count_tokens_client_version`），原有三个子测试仍 PASS。

- [ ] **Step 3: 实现 —— 扩展 endpointProtocol 映射表**

把 `handler/proxy.go` 第 18 行注释改为：

```go
// 上游协议种类。取值同时是 config.toml 中 [models.<协议>] 的段名，
// 由客户端调用的端点（见 endpointProtocol）决定，不由模型名决定。
```

把 `handler/proxy.go` 第 30-34 行的映射表改为（交给 `gofmt` 对齐）：

```go
// endpointProtocol 把客户端请求路径映射到协议种类。
// 客户端调哪个端点，就以哪种协议透传到上游同名端点。
// key 与服务端精确路径相等，不做前缀或正则匹配。
var endpointProtocol = map[string]string{
	"/v1/responses":              protocolResponses,
	"/v1/responses/input_tokens": protocolResponses, // Responses 系列的 token 计数
	"/v1/chat/completions":       protocolOpenAI,
	"/v1/messages":               protocolAnthropic,
	"/v1/messages/count_tokens":  protocolAnthropic, // Messages 系列的 token 计数
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./handler/ -run TestProxyHandler_ProtocolRouting -v`
Expected: PASS —— 6 个子测试全绿。

- [ ] **Step 5: 追加边界回归测试 `TestProxyHandler_TokenEndpointGuards`**

先在 `handler/proxy_test.go` 第 4-13 行的 import 块中补 `"strings"`（`bytes`、`encoding/json`、`net/http`、`net/http/httptest`、`testing` 与两个项目包已有）。

然后在文件末尾追加：

```go
// TestProxyHandler_TokenEndpointGuards 钉住两个 token 计数端点的边界行为：
// 非 POST 方法与路径变体必须 404、缺 model 必须 404 且不落到上游、
// 双凭证头时白名单以 Authorization 为准且客户端 x-api-key 不泄露给上游。
func TestProxyHandler_TokenEndpointGuards(t *testing.T) {
	type captured struct {
		path          string
		authorization string
		apiKey        string
	}
	hitCh := make(chan captured, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitCh <- captured{
			path:          r.URL.Path,
			authorization: r.Header.Get("Authorization"),
			apiKey:        r.Header.Get("x-api-key"),
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"input_tokens":14}`))
	}))
	defer upstream.Close()

	// drain 取出并清空已经到达上游的请求
	drain := func() []captured {
		var got []captured
		for {
			select {
			case c := <-hitCh:
				got = append(got, c)
			default:
				return got
			}
		}
	}

	cfg := &config.Config{
		AllowedKeys: []string{"client-key"},
		Platforms: map[string]config.Platform{
			"up": {BaseURL: upstream.URL, APIKey: "sk-upstream"},
		},
		Models: map[string]string{"model-a": "up"},
	}
	h := NewProxyHandler(matcher.NewMatcher(cfg))

	cases := []struct {
		name         string
		method       string
		path         string
		body         string
		authHdr      string
		apiKeyHdr    string
		wantCode     int
		wantErrMsg   string
		wantUpstream bool
	}{
		{name: "count_tokens_get_404", method: http.MethodGet, path: "/v1/messages/count_tokens", authHdr: "Bearer client-key", wantCode: http.StatusNotFound},
		{name: "input_tokens_get_404", method: http.MethodGet, path: "/v1/responses/input_tokens", authHdr: "Bearer client-key", wantCode: http.StatusNotFound},
		{name: "count_tokens_trailing_slash_404", method: http.MethodPost, path: "/v1/messages/count_tokens/", body: `{"model":"model-a"}`, authHdr: "Bearer client-key", wantCode: http.StatusNotFound},
		{name: "input_tokens_extra_segment_404", method: http.MethodPost, path: "/v1/responses/input_tokens/extra", body: `{"model":"model-a"}`, authHdr: "Bearer client-key", wantCode: http.StatusNotFound},
		{name: "count_tokens_missing_model_404", method: http.MethodPost, path: "/v1/messages/count_tokens", body: `{}`, authHdr: "Bearer client-key", wantCode: http.StatusNotFound, wantErrMsg: "model not configured"},
		{name: "input_tokens_missing_model_404", method: http.MethodPost, path: "/v1/responses/input_tokens", body: `{}`, authHdr: "Bearer client-key", wantCode: http.StatusNotFound, wantErrMsg: "model not configured"},
		{name: "count_tokens_bearer_wins_over_x_api_key", method: http.MethodPost, path: "/v1/messages/count_tokens", body: `{"model":"model-a"}`, authHdr: "Bearer client-key", apiKeyHdr: "wrong-key", wantCode: http.StatusOK, wantUpstream: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			drain()
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.authHdr != "" {
				req.Header.Set("Authorization", tc.authHdr)
			}
			if tc.apiKeyHdr != "" {
				req.Header.Set("x-api-key", tc.apiKeyHdr)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != tc.wantCode {
				t.Fatalf("expected %d, got %d (%s)", tc.wantCode, w.Code, w.Body.String())
			}
			if tc.wantErrMsg != "" && !strings.Contains(w.Body.String(), tc.wantErrMsg) {
				t.Errorf("expected body to contain %q, got %s", tc.wantErrMsg, w.Body.String())
			}

			hits := drain()
			if tc.wantUpstream {
				if len(hits) != 1 {
					t.Fatalf("expected 1 upstream request, got %d", len(hits))
				}
				if hits[0].path != tc.path {
					t.Errorf("expected upstream path %s, got %s", tc.path, hits[0].path)
				}
				if hits[0].apiKey != "sk-upstream" {
					t.Errorf("expected upstream x-api-key sk-upstream, got %q", hits[0].apiKey)
				}
				if hits[0].authorization != "" {
					t.Errorf("expected no Authorization forwarded, got %q", hits[0].authorization)
				}
			} else if len(hits) != 0 {
				t.Errorf("expected no upstream request, got %d (%+v)", len(hits), hits)
			}
		})
	}
}
```

- [ ] **Step 6: 运行边界测试确认通过**

Run: `go test ./handler/ -run TestProxyHandler_TokenEndpointGuards -v`
Expected: PASS —— 7 个子测试全绿，其中 6 个是「不落到上游」的 404 断言，`count_tokens_bearer_wins_over_x_api_key` 断言上游收到 `x-api-key: sk-upstream` 且无 `Authorization`。

- [ ] **Step 7: 跑全量测试确认没有回归**

Run: `go test ./...`
Expected: 全部包 PASS，无 FAILED。

- [ ] **Step 8: 提交**

```bash
git add handler/proxy.go handler/proxy_test.go
git commit -m "feat: 支持 count_tokens 与 input_tokens 端点透传"
```

---

### Task 2: 更新 README 端点说明

**Files:**
- Modify: `README.md:19`、`:27`、`:60`、`:108` 之后
- Modify: `README_EN.md:17`、`:25`、`:58`、`:106` 之后

**Interfaces:**
- Consumes: Task 1 落地后的端点集合（5 项）
- Produces: 无代码符号。中英文两份 README 的端点清单、协议映射表、curl 示例三处同步。

> spec 第 66 行只列了 `README.md`，但仓库里 `README_EN.md` 镜像了同一份端点清单（17/25/58/102 行）。不改它就会留下一个说「支持三种端点」的英文文档，因此本任务一并更新。

- [ ] **Step 1: 更新 README.md 特性列表（第 19 行）**

替换为：

```markdown
- **多协议** - 支持 `responses`（`/v1/responses`、`/v1/responses/input_tokens`）、`openai`（`/v1/chat/completions`）、`anthropic`（`/v1/messages`、`/v1/messages/count_tokens`）三种协议端点，请求按原格式透传到上游同名端点
```

- [ ] **Step 2: 更新 README.md 能力概述引用块（第 27 行）**

替换为：

```markdown
> **已支持五种端点**：`/v1/responses`、`/v1/responses/input_tokens`（OpenAI Responses API，Codex 使用）、`/v1/chat/completions`（OpenAI Chat Completions API）、`/v1/messages`、`/v1/messages/count_tokens`（Anthropic Messages API）。请求按原格式透传到上游同名端点，代理仅替换 Host 与鉴权头（Anthropic 自动使用 `x-api-key`）。
```

- [ ] **Step 3: 更新 README.md 协议说明（第 60 行）**

把第 60 行替换为（其余两行不动）：

```markdown
> `POST /v1/responses`、`POST /v1/responses/input_tokens` → `responses`；`POST /v1/chat/completions` → `openai`；`POST /v1/messages`、`POST /v1/messages/count_tokens` → `anthropic`。
```

- [ ] **Step 4: 追加 README.md 的 curl 示例（第 108 行代码块之后）**

```markdown

**Token 计数（Anthropic Messages API）：**
```bash
curl -X POST http://localhost:3178/v1/messages/count_tokens \
  -H "Content-Type: application/json" \
  -H "x-api-key: your-client-key" \
  -d '{"model": "claude-sonnet-4-20250514", "messages": [{"role": "user", "content": "Hello"}]}'
```

**Token 计数（OpenAI Responses API）：**
```bash
curl -X POST http://localhost:3178/v1/responses/input_tokens \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-client-key" \
  -d '{"model": "gpt-4o", "input": "Hello"}'
```
```

- [ ] **Step 5: 同步 README_EN.md（第 17、25、58 行）**

第 17 行替换为：

```markdown
- **Multi-Protocol** - Supports `responses` (`/v1/responses`, `/v1/responses/input_tokens`), `openai` (`/v1/chat/completions`), and `anthropic` (`/v1/messages`, `/v1/messages/count_tokens`) endpoints, forwarding each request unchanged to the same upstream endpoint
```

第 25 行替换为：

```markdown
> **Five endpoints are supported**: `/v1/responses`, `/v1/responses/input_tokens` (OpenAI Responses API, used by Codex), `/v1/chat/completions` (OpenAI Chat Completions API), and `/v1/messages`, `/v1/messages/count_tokens` (Anthropic Messages API). Requests are forwarded unchanged to the same upstream endpoint, with only Host and auth headers replaced (Anthropic uses `x-api-key` automatically).
```

第 58 行替换为：

```markdown
> `POST /v1/responses`, `POST /v1/responses/input_tokens` → `responses`, `POST /v1/chat/completions` → `openai`, `POST /v1/messages`, `POST /v1/messages/count_tokens` → `anthropic`.
```

- [ ] **Step 6: 追加 README_EN.md 的 curl 示例（第 106 行代码块之后）**

```markdown

**Token Counting (Anthropic Messages API):**
```bash
curl -X POST http://localhost:3178/v1/messages/count_tokens \
  -H "Content-Type: application/json" \
  -H "x-api-key: your-client-key" \
  -d '{"model": "claude-sonnet-4-20250514", "messages": [{"role": "user", "content": "Hello"}]}'
```

**Token Counting (OpenAI Responses API):**
```bash
curl -X POST http://localhost:3178/v1/responses/input_tokens \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-client-key" \
  -d '{"model": "gpt-4o", "input": "Hello"}'
```
```

- [ ] **Step 7: 校验两份 README 都已覆盖新端点**

Run: `Select-String -Path README.md, README_EN.md -Pattern '/v1/messages/count_tokens','/v1/responses/input_tokens'`
Expected: 每个文件各输出 8 行（特性列表 1、概述引用块 1、协议说明 1、curl 示例 1，各含两个路径各一次）。

- [ ] **Step 8: 提交**

```bash
git add README.md README_EN.md
git commit -m "docs: 补充 count_tokens 与 input_tokens 端点说明"
```

---

## 自查记录

- **Spec 覆盖**：接口规范（路径/鉴权/body/上游）→ Task 1 Step 1+3；响应原样透传 → Task 1 Step 3（`ReverseProxy` 未动）；错误响应不新增类型 → Global Constraints + Task 1 Step 5 钉住实际行为；端点协议表扩展 → Task 1 Step 3；注释更新 → Task 1 Step 3；README → Task 2；热重载无影响 → Global Constraints（无代码改动即覆盖）；范围外端点 → Global Constraints 明确列出。
- **步长**：每个步骤一个动作且有可检查结果；测试步骤给的是用例与断言，代码步骤给的是签名 + spec 钉死的值，无占位式描述。
- **类型一致**：`wantAnthropic` / `clientVersion` 字段在 Step 1 定义、Step 1 断言处使用；`captured` 在 Step 5 内局部声明，与 Step 1 所在函数内的同名局部类型互不干扰。
- **Proportion**：代码块集中在测试用例与文档替换文本（spec 钉死的值），实现步骤只给映射表与注释。
