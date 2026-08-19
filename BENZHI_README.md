# Structured Log Alert Aggregator

这是一个 Go 1.22 的结构化日志告警聚合器。服务接受结构化错误事件，按指纹聚合告警，并提供健康检查和告警查询接口。

## 容器构建

```sh
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh structured-log-alert-aggregator linux/arm64
./build_benzhi_docker.sh structured-log-alert-aggregator linux/amd64
docker run -it structured-log-alert-aggregator:latest
```

容器中保留完整 Go 工具链。验证命令：

```sh
go version
go build ./...
go test ./...
go vet ./...
go run ./cmd/aggregator serve
```

## 本地运行

```sh
go run ./cmd/aggregator serve
curl -fsS http://127.0.0.1:8080/v1/healthz
curl -fsS -H 'Authorization: Bearer demo-token' http://127.0.0.1:8080/v1/alerts
```

未配置数据库时，服务使用内存仓储，适用于本地演示与单元测试。

除健康检查和就绪检查外，API 必须提供 Bearer token。`API_TOKENS` 格式为逗号分隔的 `tenant:token`；未配置时仅启用开发凭据 `demo:demo-token`。

## MySQL 可选构建

MySQL 适配器位于 `internal/store/mysql.go`，使用 `mysql` 构建标签。启用前先在有网络的环境下载驱动，然后执行迁移并传入连接串：

```sh
mysql -uroot structured_alert < migrations/001_init.sql
API_TOKENS='demo:replace-with-secret' \
MYSQL_DSN='user:password@tcp(127.0.0.1:3306)/structured_alert?parseTime=true' \
  go run -tags mysql ./cmd/aggregator serve
```
