# BUG_REPRO 004：非法事件错误丢失原始 cause

## 现象

- 位置：`internal/store/memory.go` 的 `Memory.Ingest`
- 触发：`Service.Ingest` 收到一条缺字段的非法事件，返回的错误经 `errors.Unwrap` 解包后为 `nil`（错误链断裂）
- 影响：上层无法用 `errors.Is` / `errors.As` 判断底层错误类型，错误归类与重试逻辑失效

## 复现

```bash
cd structured-log-alert-aggregator
go test ./internal/app -count=1 -run '^TestBug04InvalidEventKeepsCause$'
```

- 预期：测试通过（`errors.Unwrap(err) != nil`）
- 实际：测试失败，`invalid event error lost its cause: normalize event: event_id, tenant_id, service, environment and message are required`

## 根因线索

- `memory.go` 中 `fmt.Errorf("normalize event: %v", err)` 使用了 `%v` 而不是 `%w`，未把原始错误包装进错误链
- `%w` 才能让 `errors.Unwrap` 解出 cause；`%v` 只做格式化输出
- 修复：将 `%v` 改为 `%w`

## 验证命令（修复后应为绿）

```bash
go test ./internal/app -count=1 -run '^TestBug04InvalidEventKeepsCause$'
```
