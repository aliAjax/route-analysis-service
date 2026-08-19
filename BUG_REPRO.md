# BUG_REPRO — route-analysis-service__009

## Bug 是什么
查询两组重叠事件的冲突列表时，第一组的边被第二组覆盖，响应里两组显示同一条边。

## 如何触发
运行 `go test ./internal/r9 -run '^TestIncidentConflictsKeepOwnEdgeLists$' -count=1`。

## 错误信息
第一组 conflicts[0].edgeIds 变成 bc 而不是 ab。
