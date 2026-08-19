# BUG_REPRO — route-analysis-service__003

## Bug 是什么
生成网络质量报告再导出 CSV 时，network 节内容被 modes 节覆盖；连续两次汇总的 mode 列表互相污染。

## 如何触发
运行 `go test ./internal/r3 -run '^TestBuildNetworkReportCSVAndSummariseIndependent$' -count=1`。

## 错误信息
network 节首行变成 mode，CSV 缺 metric/toll_edges，SummariseDataset 的 Modes 前后互相串。
