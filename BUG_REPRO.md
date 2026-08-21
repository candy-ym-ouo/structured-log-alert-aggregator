# BUG_REPRO 005：Recovery 未透传 context 取消信号

## 现象

- 位置：`internal/worker/recovery.go` 的 `Recovery.Once`
- 触发：以已取消的 `ctx` 调用 `Recovery.Once`，仓库层收到的 context 并未被取消（被替换成了 `context.Background()`）
- 影响：取消/超时信号无法下传，长时间运行的恢复任务无法被及时终止，可能造成资源占用与重复处理

## 复现

```bash
cd structured-log-alert-aggregator
go test ./internal/worker -count=1 -run '^TestBug05RecoveryForwardsCancellation$'
```

- 预期：测试通过（`repo.seen.Err() == context.Canceled`）
- 实际：测试失败，`repository context was not canceled: <nil>`

## 根因线索

- `recovery.go` 中 `w.Repo.DueForRecovery(context.Background(), now)` 未透传入参 `ctx`，取消信号在入口处即被丢弃
- 修复：改为 `w.Repo.DueForRecovery(ctx, now)`
- 另注意 `memory.go` 的 `DueForRecovery` 中 `ctx = context.Background()` 同样吞掉了调用方传入的取消信号，属于同类问题

## 验证命令（修复后应为绿）

```bash
go test ./internal/worker -count=1 -run '^TestBug05RecoveryForwardsCancellation$'
```
