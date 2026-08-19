# BUG_REPRO — route-analysis-service__001

## Bug 是什么
事件激活接口把「不存在的事件」和「If-Match 版本冲突」都映射成 400 operation_failed，而不是 404 not_found 和 409 conflict。

## 如何触发
1. 启动服务：`go run ./cmd/route-service`
2. 激活一个不存在的事件：
   `POST /api/v1/incidents/inc-missing/activate?datasetId=demo-city`，请求头 `If-Match: 1`
3. 或先创建事件，再用过期的 `If-Match` 版本去激活同一事件。

## 错误信息
不存在事件时返回：
```json
{"error":{"code":"operation_failed","message":"incident inc-missing: resource not found"}}
```
版本冲突时返回：
```json
{"error":{"code":"operation_failed","message":"activate incident inc-1: incident inc-1: resource conflict"}}
```
正确行为应分别是 404 not_found 与 409 conflict。
