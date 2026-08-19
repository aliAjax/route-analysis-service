# BUG_REPRO — route-analysis-service__008

## Bug 是什么
草稿数据集未导入图也能通过校验，非法状态不报错，导入完成的迁移被拒绝。

## 如何触发
运行 `go test ./internal/r8 -run '^TestDatasetStateMachineGatesValidation$' -count=1`。

## 错误信息
DatasetTransition(importing, ready) 为 false，草稿状态 ValidateDatasetState 返回 nil。
