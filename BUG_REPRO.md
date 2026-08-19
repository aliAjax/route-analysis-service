# BUG_REPRO — route-analysis-service__007

## Bug 是什么
并发批量校验数据集时产生 data race，返回结果可能缺条目。

## 如何触发
运行 `go test -race ./internal/r7 -run '^TestValidateDatasetsBatchConcurrentNoRace$' -count=1`。

## 错误信息
WARNING: DATA RACE，位于 worker/dispatcher.go 与 application/batch_service.go。
