# 城市树木健康巡检与修复验收台

本项目是面向城市绿化团队的本地浏览器工作台，将树木档案登记、巡检证据、风险评估、修复派发、现场处置、复验关闭和养护凭据串成一条可追溯流程。业务数据保存在 SQLite 中，关键写入同时记录审计哈希链和恢复检查点；写入 API 支持幂等重试与 `expectedVersion` 冲突保护。

## 构建与运行

```bash
go build ./...
go run .
```

默认地址为 `http://127.0.0.1:19081`，默认数据库为 `data/city-tree.db`。可通过 `-addr` 指定回环地址，也可通过 `PORT` 指定端口；例如：

```bash
go run . -addr=127.0.0.1:19090 -db=data/city-tree.db
```

服务只接受 `127.0.0.1`、`localhost` 或 `::1` 回环监听地址。浏览器打开 `/` 即可完成主要业务流程，同源 JSON API 位于 `/api`。

## 测试与自检

```bash
go test ./...
go run . -selfcheck -addr=127.0.0.1:19082
```

`-selfcheck` 使用临时 SQLite 数据库，真实启动指定回环 HTTP 服务，依次创建批次、提交证据、评估风险、确认修复并复验签证，同时验证幂等结果复用、版本冲突保护和养护凭据摘要，完成后自行退出。

## 数据与接口约定

所有写入请求必须携带 `Idempotency-Key` 请求头。涉及已有树木或任务的更新还需在 JSON 中提交页面读取到的 `expectedVersion`；版本落后时 API 返回 `409 Conflict`，不会覆盖较新的现场记录。

SQLite 使用 WAL 与 `synchronous=FULL`。数据库旁的 `.integrity` 文件是原子写入的审计尾检查点，服务重启时会校验数据库关键写入是否与检查点一致。
