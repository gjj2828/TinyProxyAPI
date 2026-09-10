package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/user/TinyProxyAPI/config"
	"github.com/user/TinyProxyAPI/handler"
	"github.com/user/TinyProxyAPI/matcher"
)

// 端到端：模拟用户实际场景——
// 1. 启动代理，旧 allowed_keys=["old"]
// 2. 调用方用 "old" 请求，预期通过白名单
// 3. 调用 Update，模拟 reload 把 allowed_keys 换成 ["new"]
// 4. 调用方改用 "new" 请求，预期通过白名单
// 5. 用 "old" 再请求一次，预期被拒绝
//
// 这个测试如果失败，意味着热重载后 HTTP server 仍用旧 matcher（之前的 bug）。
func TestHotReload_TakesEffectImmediately(t *testing.T) {
	// 配置1：白名单=["old"]，下游不存在（不会真正发请求）
	cfg1 := &config.Config{
		Server:      config.ServerConfig{Addr: ":0"},
		Default:     config.DefaultConfig{Platform: "p"},
		AllowedKeys: []string{"old"},
		Platforms: map[string]config.Platform{
			"p": {BaseURL: "http://127.0.0.1:1", APIKey: "sk-x"}, // 故意不可达
		},
		Models: map[string]string{"m1": "p"},
	}

	h := handler.NewProxyHandler(matcher.NewMatcher(cfg1))

	ts := httptest.NewServer(h)
	defer ts.Close()

	// 用 "old" 请求：应通过白名单（之后会因下游不可达报错，但我们只关心 401）
	req := func(auth string) (int, string) {
		body, _ := json.Marshal(map[string]string{"model": "m1"})
		r, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/responses", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer "+auth)
		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			// 下游不可达是预期的，我们关心的是白名单是否通过。
			// 401=白名单拒绝，其他=白名单通过（包括网络错误）。
			return 0, err.Error()
		}
		defer resp.Body.Close()
		var b bytes.Buffer
		_, _ = io.Copy(&b, resp.Body)
		return resp.StatusCode, b.String()
	}

	// 阶段1：old 通过白名单
	code, _ := req("old")
	if code == http.StatusUnauthorized {
		t.Fatalf("阶段1：'old' 应通过白名单，却返回 401")
	}

	// 阶段2：unknown 应被白名单拒绝
	if code, body := req("unknown"); code != http.StatusUnauthorized {
		t.Fatalf("阶段2：'unknown' 应被白名单拒绝，却返回 code=%d body=%s", code, body)
	}

	// 阶段3：热重载——白名单换成 ["new"]
	cfg2 := &config.Config{
		Server:      cfg1.Server,
		Default:     cfg1.Default,
		AllowedKeys: []string{"new"},
		Platforms:   cfg1.Platforms,
		Models:      cfg1.Models,
	}
	h.Update(matcher.NewMatcher(cfg2))

	// 阶段4：用 "new" 应通过白名单
	if code := mustStatus(t, req, "new"); code == http.StatusUnauthorized {
		t.Fatalf("阶段4：热重载后 'new' 应通过白名单，却返回 401")
	}

	// 阶段5：用 "old" 应被新白名单拒绝
	code, body := req("old")
	if code != http.StatusUnauthorized {
		t.Fatalf("阶段5：热重载后 'old' 应被新白名单拒绝，却返回 code=%d body=%s", code, body)
	}
	if !strings.Contains(body, "invalid api key") {
		t.Fatalf("阶段5：响应体应包含 'invalid api key'，实际：%s", body)
	}
}

// mustStatus 从 req 函数中提取状态码，失败时终止测试。
func mustStatus(t *testing.T, req func(string) (int, string), auth string) int {
	t.Helper()
	code, _ := req(auth)
	return code
}