# BUG_REPRO — route-analysis-service__010

## Bug 是什么
聚类后两个簇的节点列表互相污染，连续两次距离计算的结果互相覆盖。

## 如何触发
运行 `go test ./internal/r10 -run '^TestClusterAndDistanceResultsDoNotAlias$' -count=1`。

## 错误信息
修改第一个簇节点后第二个簇跟着变，第一次距离列表被第二次覆盖。
