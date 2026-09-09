# Task 4 Report: 单元测试更新

## Status: DONE

## Commit

- **SHA**: fcad3ef5e20c9e75fdcaea8645a5d43798bf9abd
- **Subject**: test: add tests for API key whitelist feature

## Test Results Summary

All tests pass:

```
=== RUN   TestLoadConfig
--- PASS: TestLoadConfig (0.01s)
PASS

=== RUN   TestProxyHandler_ServeHTTP
--- PASS: TestProxyHandler_ServeHTTP (0.00s)
PASS

=== RUN   TestMatcher_Match
--- PASS: TestMatcher_Match (0.00s)
=== RUN   TestMatcher_GetAllowedKeys
--- PASS: TestMatcher_GetAllowedKeys (0.00s)
PASS

=== RUN   TestWatch
--- PASS: TestWatch (0.10s)
PASS
```

## Changes Made

### 1. config/loader_test.go
- Added `allowed_keys` to test TOML content (must be declared before `[section]` blocks)
- Added assertion for `AllowedKeys` parsing (3 keys expected)

### 2. matcher/matcher_test.go
- Added `TestMatcher_GetAllowedKeys` function to test `GetAllowedKeys()` method

### 3. handler/proxy_test.go
- Updated config to include `AllowedKeys` for auth testing
- Added test for 401 when Authorization header is missing
- Added test for 401 when API key is invalid

### 4. config.toml.example
- Added `allowed_keys = ["your-key-here"]` section

### 5. config/loader.go (bug fix)
- Added `toml:"allowed_keys"` struct tag for correct field mapping

## Key Finding

go-toml/v2 requires:
1. Top-level values must be declared BEFORE any `[section]` blocks
2. Explicit `toml` struct tag for snake_case to PascalCase field mapping
