# BUG_REPRO — route-analysis-service__005

## Bug 是什么
解决过期事件时返回 id 列表多出未过期事件，且未过期事件版本被改动；批量迁移 Updated 列表包含失败项。

## 如何触发
运行 `go test ./internal/r5 -run '^TestResolveExpiredOnlyResolvesExpiredIncidents$' -count=1`。

## 错误信息
返回 ids 含 inc-future 与重复项，BatchTransition 的 Updated 含失败项。
