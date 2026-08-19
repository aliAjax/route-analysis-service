# BUG_REPRO — route-analysis-service__006

## Bug 是什么
并发读取同一缓存路由时产生 data race，偶尔 panic。

## 如何触发
运行 `go test -race ./internal/r6 -run '^TestCachedRouteConcurrentReadsNoRace$' -count=1`。

## 错误信息
WARNING: DATA RACE，位于 cache/lru.go 与 application/route_service.go。
