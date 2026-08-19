# BUG_REPRO — route-analysis-service__002

## Bug 是什么
注册租户配额策略后授权请求时，评估器写入 nil map 导致 panic。

## 如何触发
运行 `go test ./internal/r2 -run '^TestEvaluatorRegisterDoesNotPanic$' -count=1`，或调用 Register 后再 Authorize。

## 错误信息
panic: assignment to entry in nil map [recovered, repanicked]，栈顶位于 internal/domain/quota/quota.go Register。
