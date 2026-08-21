# BUG_REPRO 003：策略通道在 Ingest 过程中被篡改

## 现象

- 位置：`internal/domain/policy.go` 的 `SelectPolicy`（配合 `internal/store/memory.go` 的 `Policies`）
- 触发：`Service.Ingest` 处理一条日志事件后，预置的 `AlertPolicy.Channels` 列表被改写为 `["audit"]`
- 影响：告警策略的通道配置被意外篡改，后续通知会发往错误渠道

## 复现

```bash
cd structured-log-alert-aggregator
go test ./internal/store -count=1 -run '^TestBug03PolicyChannelsAreNotMutatedDuringIngest$'
```

- 预期：测试通过（`Channels` 保持 `["email"]`）
- 实际：测试失败，`policy channels changed to []string{"audit"}`

## 根因线索

- `SelectPolicy` 中 `ps[0].Channels[0] = "audit"` 直接修改了传入切片底层数组中元素的内容
- Go 切片按引用传递，`Memory.Policies` 返回的切片与内部 `m.policies` 共享底层数组，任何外部写入都会污染内部状态
- 该赋值语句无任何业务含义，属于被注入的缺陷，应删除

## 验证命令（修复后应为绿）

```bash
go test ./internal/store -count=1 -run '^TestBug03PolicyChannelsAreNotMutatedDuringIngest$'
```
