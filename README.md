# Structured Log Alert Aggregator

生产型结构化日志告警聚合器的最小可运行实现。设计依据见附件规格文档。

## 运行

```sh
go test ./...
go run ./cmd/aggregator serve
```

服务默认监听 `:8080`，提供 `/v1/healthz`、`/v1/readyz` 与事件摄取接口。

## MySQL

数据库目标为 MySQL 8.0+。执行 `migrations/001_init.sql` 后，用 DSN 配置数据库：

```sh
MYSQL_DSN='user:password@tcp(127.0.0.1:3306)/alerts?parseTime=true' \
  go run -tags mysql ./cmd/aggregator serve
```

未设置 `MYSQL_DSN` 时使用内存仓储，适合本地接口演示和单元测试。
