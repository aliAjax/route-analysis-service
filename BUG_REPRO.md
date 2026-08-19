# BUG_REPRO — route-analysis-service__004

## Bug 是什么
请求取消后 NearestNode 仍继续遍历全图并返回节点，而不是返回取消错误。

## 如何触发
运行 `go test ./internal/r4 -run '^TestNearestNodeRespectsCancellation$' -count=1`，使用取消中的 context。

## 错误信息
取消后仍返回节点与距离，未返回 context.Canceled。
