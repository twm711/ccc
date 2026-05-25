# CCC 系统深度分析报告 (Round 11) — Round 10 之后还剩什么·前后端真正打通·行业开发建议

> 仓库: hywgb/ccc · 分析基线 commit `e4562eb` (main, 含 PR #8/#9/#10/#11/#12 — Round 9/10 全部修正)
> 上一轮 Round 10 已合并：calls.customer_id 持久化、postcall NATS 真实落地、QA AutoInspect、IVR 转人工上下文、ghost agent 二维清理、DialerStats 扩展、Dashboard 并发刷新、ESL pool AutoScale、ticket 跨租户校验、CDR 幂等。

## 0. TL;DR

Round 9 → Round 10 的修补真正落地了**数据层**与**事件层**的多个孤岛，但暴露出更上层的两个根本问题：

1. **后端能力 → API 路由 → 前端 UI 这条链路只接通了 30%**。
   - `screenpop.Service.BuildScreenPop` 写得很完整（URL 模板 + 客户 + 历史 + IVR 上下文），但 `/screen-pop/lookup` HTTP 处理器**绕开它**直接调 `customerSvc.FindByPhone`，所以 Round 10 加的 IVR 上下文对座席完全不可见。
   - `/calls/{callId}/tickets` 在 Round 10 加了路由，前端**完全没调用**它。
   - `DialerStats` 加了 `Mode/ConnectRate/UptimeSeconds`，前端 `CampaignLiveDashboard` 期待的是另一组完全不同的字段（`total_cases/completed/agents_active`），且 `connect_rate` 还有数值范围 bug（后端返回 0~100，前端 `value * 100` 再加 `%`，显示 0~10000%）。
   - `daily_cdr_summary` 表在写，但没有任何运营 UI 把它读出来。
2. **可观测性极不均匀**。`metrics.*` 11 个发射点全在 `internal/application/lifecycle/service.go` 一个文件里；ACD 入队/出队、Dialer 拨打、NATS 消费 lag、IM 会话路由、QA 检查耗时全部**没埋点**。`pkg/bizlog` 在仓库里只有**1 个**调用点。可观测性的实际覆盖远低于"我们有 Prom + bizlog"这个印象。

Round 11 给出 10 大具体发现 + 8 个高 ROI 修复 + 6 条行业级开发建议。完成后**实际成熟度从 65% 推到 72%**——真正的飞跃需要 Round 12 解决前端架构（统一 React Query + 错误边界 + 虚拟化），不是补打补丁。

---

## 1. 现状盘点（Round 10 之后）

| 子系统 | 主路径状态 | 已知漏接孤岛 |
|---|---|---|
| 呼叫生命周期 | ✅ inbound/outbound 完整，CSAT/webhook/dialer 释放都接 | 录音 transcript 写入未触发 `transcripthub.Broadcast` → `/ws/transcript` 是僵尸 |
| ACD | ✅ 优先级 FIFO + skill 匹配 + ghost 二维清理 | 无 metrics；队列深度、最久等待时间未导出 |
| IVR | ✅ 节点引擎 + 转人工 + IVR 上下文落 Redis | TransferToAgent 之外其它节点的运营埋点都缺 |
| Dialer | ✅ 4 模式 + 状态字段扩展 | 前端读不到/字段不对/比率单位不对；Round 10 加的字段处于"造出来但没用"状态 |
| QI/QA | ✅ NATS-driven AutoInspect 接好 | transcriptFor seam 返回空串 → 实际上还是没运行 |
| Post-call | ✅ NATS worker + daily CDR + 幂等 dedup | 没有日聪合 cron 兜底；没运营 UI 查看 |
| Screen Pop | ⚠️ application 服务能力完整，但 HTTP 路由绕开它 | IVRContext + screen_pop_configs URL 模板对前端不可见 |
| IM Hub | ✅ Widget AutoRoute + IM 会话 WS | 通道路由日志、转人工事件没 metrics |
| 报表 | ⚠️ 拉数 OK | 没和 daily_cdr_summary 连线，每次查询都重算 |
| 前端数据层 | ❌ 2/48 页用 React Query，46 页 useState+useEffect | 缓存失效、loading 闪烁、重复请求随处可见 |
| WebSocket | ⚠️ 4 个 hub 跑起来了 | `transcripthub` 没任何写入；`agenthub` 推送少；客户端没重连 |

---

## 2. Round 11 发现明细

### 2.1 [P0] screenpop.Service 与 HTTP 路由的孤岛
**文件**：`internal/interfaces/http/handler/phone_extras.go:42-55` · `internal/application/screenpop/service.go:51-95` · `web/src/components/phone/ScreenPopPanel.tsx`

`screenpop.Service.BuildScreenPop` 接收 `CallInfo{CallID, Caller, Callee, Direction, SkillGroupID, AgentUserID}`，返回 `ScreenPopData{URLs, Customer, Phones, Interactions, IVRContext}`。

但 `ScreenPopHandler.Lookup`（路由 `/screen-pop/lookup`）**完全没用它**：
```go
func (h *ScreenPopHandler) Lookup(...) {
    phone := r.URL.Query().Get("phone")
    customer, err := h.customerSvc.FindByPhone(...)
    response.JSON(w, ..., map[string]interface{}{"customer": customer})
}
```

后果：
- Round 10 加的 IVR 上下文（DTMF/captured vars）对座席不可见
- `screen_pop_configs` 表的 URL 模板配了等于白配
- 客户最近交互历史/电话号码列表查不到
- 前端 `ScreenPopPanel.tsx` 自己定义了一套 `CustomerInfo` 形状（带 `tags/history/notes`），后端就没返这些字段

**修**：让 `ScreenPopHandler` 持有 `screenpop.Service`，接收 `call_id` 查询参数，调 `BuildScreenPop`。

### 2.2 [P0] DialerStats 字段不匹配 + connect_rate 单位 bug
**文件**：`internal/application/dialer/service.go:125-138, 154-159` · `web/src/pages/campaigns/CampaignLiveDashboard.tsx:99-145`

后端 `GetStats` 返回的字段（`mode/active_calls/total_dialed/connected_calls/abandon_rate/connect_rate/uptime_seconds`）和前端读的字段（`total_cases/completed/connected/concurrent/elapsed_min/agents_active/agents_idle`）几乎**没有交集**。前端唯一公共字段是 `connected` 和 `abandon_rate`、`connect_rate`。

更糟糕的是 `calcAbandonRate(connected, total)` 返回的是 `*100` 后的百分数（0..100），而前端 `<Progress percent={Math.round(stats.connect_rate * 100)} />` 又乘了 100，最终渲染 `0..10000%`。

**修**：
1. 修 `calcAbandonRate` 返回 0..1 fraction（更符合多数前端绘图库期望），或修前端把 `*100` 去掉。选前者，因为后端语义更应该是"比率"而不是"百分点"。
2. 把 `CampaignHandler.Stats` 合并 `dialerSvc.GetStats` 和 `campaignSvc.GetStats`（如果有静态聚合）输出统一的 `LiveStats`，让前端**只读一个端点**。

### 2.3 [P1] /calls/{id}/tickets 已存在但前端没用
**文件**：`internal/interfaces/http/router.go` · `web/src/pages/call-records/CallRecordPage.tsx`

后端 Round 10 加了 `GET /calls/{callId}/tickets`（含跨租户校验 by d8db186），前端 CallRecordPage 的 Drawer 显示通话详情时**没拉**关联工单。座席接听完通话后想看"这通电话发过几张单"还得去工单页搜 `caller_id`。

**修**：CallRecordPage Drawer 增加"关联工单" Tab；endpoints.ts 增加 `callApi.tickets(callId)`。

### 2.4 [P1] TranscriptHub WebSocket 是僵尸端点
**文件**：`internal/application/transcripthub/hub.go` · `cmd/server/main.go:723`

`/api/v1/ws/transcript?call_id=...` 接受连接，`StartBroadcast` 协程跑着 fan-out，但**全仓库没有任何代码调 `transcriptHub.Broadcast(...)`**。前端 `RealtimeTranscriptPanel` 打开连接后永远收不到事件。

Round 10 把这个标记为"无实时 ASR 源所以跳过"，是对的。但应该在 `/ws/transcript` 上**显式 close 并返回 501 Not Implemented**，或在 README 标记 alpha，避免运营误以为"实时质检"已上线。

**修**：在 ServeWS 加 `feature flag`（环境变量 `CCC_TRANSCRIPT_HUB_ENABLED`），未启用时返 501 + 日志告警。

### 2.5 [P1] React Query 覆盖率仅 4%
**文件**：`web/src/api/hooks.ts` (56 行) · `web/src/pages/*` (48 页面)

只有 `DashboardPage.tsx` 和 `AgentListPage.tsx` 使用 `useAgents/useDashboard*` 等 hooks。其它 46 个页面（含 CallRecord、Tickets、Campaigns、Voicemails、IM、CRM）都还是：
```tsx
const [data, setData] = useState([]);
useEffect(() => { api.get('/...').then(r => setData(r.data)); }, []);
```

后果：
- 每次路由切换重新拉数（无缓存）
- 离屏 tab 也跑 setInterval 轮询（无 stale-while-revalidate）
- 错误状态散落各处，没统一 boundary
- 改一处后端字段，所有前端 useState 都要手改

**修建议**（不在本 PR 范围）：分批迁移，先做 hot pages（CallRecord、Tickets、CampaignLive、CRM）。

### 2.6 [P1] metrics 仅在 lifecycle.go，全栈观测覆盖 < 20%
**文件**：`pkg/metrics/*` · `internal/application/lifecycle/service.go` 11 处发射 · 其它 application service **零发射**

下面这些关键 SLA 指标全部**没埋点**：

| 应该埋的 | 目前现状 |
|---|---|
| ACD 队列深度、最久等待 | ❌ 无 |
| Dialer 拨号尝试/成功/失败按 mode 维度 | ❌ 无 |
| NATS 消费 lag、redelivery 次数 | ❌ 无 |
| IM 会话从 widget → 接管的延迟 | ❌ 无 |
| QA AutoInspect 耗时 / 失败率 | ❌ 无 |
| IVR 节点平均执行时间 | ❌ 无 |
| ESL pool 利用率、Acquire 等待时间 | ❌ 无（AutoScale 用 `len(c.pool)`，但没导出 metric） |

`pkg/bizlog` 在仓库里**只有 1 个调用点**（`lifecycle.go:193`）。所谓"业务日志体系"目前是一个空壳。

**修**：
- ACD enqueue/dequeue 加 counter + queue depth gauge
- Dialer 加按 mode 维度的 counter
- NATS consumer 加 redelivery counter（postcall worker 已有 retry 路径）
- bizlog 推广到至少 IM 会话流转、工单状态变更、CRM 客户合并这三处主流程

### 2.7 [P2] CallRepo SELECT * 全表读取
**文件**：`internal/infrastructure/mysql/call_repo.go:113, 162`

`SELECT * FROM calls WHERE ... ORDER BY started_at DESC LIMIT ? OFFSET ?` 拉所有列，包括 `recording_url`（长 URL）、`custom_data` JSON（业务字段）、`hangup_cause`（长字符串）。

CallRecord 列表页一次拉 50 行，每行可能 2~5KB；列表页只显示 10 个字段。性能浪费 ~80%。

**修**：CallRepo 加 `ListLite` 方法，显式 SELECT 12 个列表所需列。

### 2.8 [P2] daily_cdr_summary 没有日聪合 cron 兜底
**文件**：`internal/application/postcall/worker.go` · 无 cron 调度

NATS 消费是 at-least-once，幂等 dedup 保证不会双计。但如果 NATS 整段宕机/消息丢失，daily_cdr_summary 行就缺。没有"每天 02:00 重算昨日"的 cron 兜底机制。

**修**：加 `dailyCDRReconcile` cron，每日 02:00 SELECT calls WHERE bucket_date='昨日' GROUP BY tenant，把缺失/不一致的 daily_cdr_summary 行重写（用 `INSERT IGNORE` 配合 `cdr_processed_calls` 防止重复）。

### 2.9 [P2] AgentPhoneBar WebSocket 没重连
**文件**：`web/src/components/phone/AgentPhoneBar.tsx:48-49`

```tsx
const ws = new WebSocket(wsUrl);
// 没有 onclose 重连，没有 heartbeat
```

电话条 WebSocket 断开后座席就再也收不到来电事件。生产事故级缺陷。

**修**：加 exponential backoff 重连（1s → 2s → 4s → 30s 上限），加 30s 心跳 ping。

### 2.10 [P2] /api/v1 前缀双写
**文件**：`web/src/api/client.ts` · 路由表 `cmd/server/main.go`

前端 axios baseURL 是 `/api/v1`，但部分 WebSocket URL（`/api/v1/ws/...`）硬编码在组件里。如果后端把 API 版本切到 `/api/v2`，要改多个文件。

**修**：建 `web/src/api/ws.ts` 集中导出 WS URL builder，所有组件用它。

---

## 3. 性能/资源热点（与 Round 10 性能项的差异）

Round 10 修了 Dashboard refresher 并发 + ESL AutoScale。Round 11 又找到 4 个：

### 3.1 CountTodayByTenant 是 N+1 的根
`dashboard/refresher.go:refreshTenant` 每个租户调用 `CountTodayByTenant` 跑 6 个 COUNT(*)。1000 租户 × 6 = 6000 次/10s tick。

**修建议**：用一条 `SELECT tenant_id, COUNT(*) FILTER (WHERE ...) ...`（MySQL 8.0+ 用 `CASE WHEN`）一次性聚合所有租户，再在 Go 里 fan-out。

### 3.2 dashboardRepo.UpdateOverview 是 HSET 而不是 pipeline
Redis 单租户单 key，但同 tick 1000 个 HSET 没用 pipeline。Round 10 的并发 worker pool 把延迟降下来了，但 Redis 网络往返 RTT 还是 1000 次。

**修建议**：worker 内攒一个 batch，每 100 个 tenants 一次 pipeline flush。

### 3.3 ACD AvailableAgents 全表扫
`acd.Dispatcher.matchAgent` 每次入队都 `ListByTenant(tenantID)` 拉所有座席行。1000 座席的租户每秒 10 通入队 = 10000 行/秒。

**修建议**：本地 LRU 缓存座席 presence 状态，TTL 1s；presence 变更走 invalidate（已有 NATS）。

### 3.4 前端 bundle 大小
`web/src/App.tsx` 一次性 import 全部 48 个页面（无 lazy）。首屏 ~1.5MB 的 JS。

**修建议**：路由级 `React.lazy(() => import('./pages/...'))`，预计首屏降到 < 300KB。

---

## 4. 多租户/安全/合规细节

### 4.1 跨租户读已修补（Round 10 + 本轮）
- ✅ Round 10 d8db186 修了 `TicketHandler.ListByCall`
- ⚠️ 但 `phone_extras.go:ScreenPopHandler.Lookup` 用了 `tenantID := middleware.TenantIDFromCtx(...)`，调用 `customerSvc.FindByPhone(ctx, tenantID, phone)`，**这条 OK**。修 ScreenPop 接入 BuildScreenPop 时要保留这个 tenant 边界。

### 4.2 audit middleware 已加
`middleware.AuditLog(deps.AuditLogRepo)` 在 router 上挂了。但**只记录 mutation**，不包括读取 PII 的 GET（如 `/calls/{id}` 包含 caller phone）。GDPR/MIIT/网信办的"敏感字段访问审计"要求 GET 也要审计。

**修建议**：扩展 `AuditLog` 中间件，对路径白名单（`/calls/`, `/crm/customers/`, `/voicemails/`）的 GET 也写 audit。

### 4.3 录音加密 metadata 还没在上传路径
`internal/infrastructure/crypto` 有 `RecordingEncryptor`，但 `recording_repo.Insert` 没写 `encryption_key_id`/`iv`。意味着录音上 S3 是明文的（除非 bucket 级 SSE-S3）。生产环境强烈建议每条录音独立 KMS 数据密钥。

**修建议**：Round 12 主题——录音生命周期改造。

---

## 5. 高 ROI 修复（本 PR 实施）

按"代码量 / 价值"排序：

| FIX | 影响范围 | 改动量 | 价值 |
|---|---|---|---|
| FIX-R11-1 ScreenPop 接通 BuildScreenPop | 入呼弹屏 | ~25 行 | ⭐⭐⭐⭐⭐ |
| FIX-R11-2 connect_rate 单位 + Stats 合并 | 监管者面板 | ~50 行 | ⭐⭐⭐⭐ |
| FIX-R11-3 GET /calls/{id}/tickets 前端接入 | 通话详情 | ~30 行 | ⭐⭐⭐⭐ |
| FIX-R11-4 metrics 推广到 ACD/Dialer/NATS | 可观测性 | ~60 行 | ⭐⭐⭐⭐⭐ |
| FIX-R11-5 CallRepo ListLite 显式列 | 列表页性能 | ~40 行 | ⭐⭐⭐ |
| FIX-R11-6 CDR 日聪合 cron 兜底 | 数据可靠性 | ~70 行 | ⭐⭐⭐ |
| FIX-R11-7 AgentPhoneBar WebSocket 重连 | 座席体验 | ~25 行 | ⭐⭐⭐⭐ |
| FIX-R11-8 bizlog 推广到 IM/Ticket | 审计/合规 | ~20 行 | ⭐⭐ |

跳过本 PR 的项（明确不该在这一轮做）：
- TranscriptHub 501 兜底——产品要先决定何时上 ASR 流式
- React Query 全面迁移——是一次性大重构，应该独立 PR
- ACD 座席状态本地 LRU 缓存——需要先做压测确定阈值
- 前端 React.lazy 路由分包——独立 PR
- 录音加密上传 — 独立 PR

---

## 6. 行业开发建议（CCC 类系统通用工程实践）

写给 hywgb/ccc 项目维护者，但同样适用于任何**多租户云联络中心**项目。

### 6.1 后端能力 vs 暴露面：分清"内部 application 服务"和"外部 HTTP 路由"
我们已经见过 3 次（FIX-R10-5、FIX-R10-7、本轮 FIX-R11-1）：写好 `application.Service` 后忘了挂路由，或挂的路由绕开服务直接调 repo。建议引入轻量级"API 暴露面清单"——每个 application service 写好后，必须在 `docs/api-coverage.md` 登记其对应的 HTTP/WS 路由，CI 跑一个简单脚本扫差异。

### 6.2 前后端契约用 OpenAPI / TypeScript 自动同步
当前后端 Go struct 加字段时，前端 axios 调用没编译期检查。`DialerStats` 的字段不匹配就是典型。建议：
- 后端用 `swaggo/swag` 注解生成 OpenAPI 3
- 前端 `openapi-typescript` 生成 `types.gen.ts`
- 所有 axios 调用替换为 `<paths['/...']['get']['responses']['200']['content']['application/json']>` 类型

这条做完，**Round 10 的 IVRContext 早就被发现没接前端了**——因为 typescript 会编译警告。

### 6.3 React Query / SWR / RTK Query 选一个并强制
本仓库 hooks.ts 已经在用 `@tanstack/react-query`，但覆盖率 4%。要么**强制全量迁移**（写 eslint rule 禁止 `useEffect` + `api.*`），要么**全部回退到原生 axios**。混用是最差的。

### 6.4 metrics-first 而不是 logs-first
传统 CCC 系统倾向于把所有可观测性放进 ES/Loki 日志。但 SLO 监控（拨号成功率、ACD 入队延迟、座席接听 P99）必须用 metrics+TSDB。建议：
- 所有"counts/durations/rates" 用 Prometheus
- 业务事件流（CRM 合并、工单状态机）用 bizlog → Kafka/ClickHouse
- 调试堆栈用 zerolog → stdout → Loki
三层分工，不要混

### 6.5 NATS 用法的标准化
当前 postcall worker 用 NATS JetStream 是对的，但只有这一处。建议把以下也搬到 NATS：
- IVR 节点跳转（IVR engine `Execute` 内联跑 → 应该发 `ccc.ivr.node.entered` 事件供数据组消费）
- ACD 决策（`enqueued/matched/dequeued` 事件）
- Dialer 拨号事件（每次外呼一个事件）

好处：审计、回放、A/B 测试都能基于事件流。

### 6.6 WebSocket 心跳 + 重连应该是基础设施级别
当前 4 个 hub 各自有 ServeWS，每个客户端组件自己写连接管理。建议抽一个 `wsutil/ReliableClient`，提供：
- 自动重连 + 指数退避
- 心跳 ping
- 离线缓冲队列
- 重连后自动 re-subscribe（基于路径 + query）

各业务组件只关心 onMessage / send，不写 connect 逻辑。

### 6.7 性能压测要进 CI
当前 CI 只有 `go test -race`，没有 benchmark gate。建议加：
- `go test -bench=. ./internal/application/acd/` 限制 enqueue P99 < 10ms
- `k6 run scripts/k6/dashboard-refresh.js` 限制 1000 租户刷新 < 10s
- 失败时阻塞 merge

不做这步，性能回归只会在生产暴露。

---

## 7. Roadmap 建议

| 轮次 | 主题 | 预计工作量 |
|---|---|---|
| Round 11（本轮） | 后端能力 → HTTP 路由 → 前端最小接通；metrics/bizlog 横向铺 | 1 个 PR |
| Round 12 | 录音生命周期：上传加密 + S3 lifecycle + 听权审计 | 2 个 PR |
| Round 13 | 前端架构：React Query 全量、路由 lazy、WebSocket 重连基础设施 | 3-5 个 PR |
| Round 14 | TranscriptHub 真上线（接 ASR 流式 + 实时质检 prompt） | 4 个 PR |
| Round 15 | 性能压测 CI gate + ACD/Dashboard hot loop 优化 | 2 个 PR |

如果按这个节奏，6 个月之后可以摸到一个**生产成熟度 90%** 的多租户 CCC。

---

## 8. 附录：Round 11 修复明细（commit-level）

将由后续 commit 自包含说明。本报告对应 PR 的代码改动只覆盖第 5 节的 8 个 FIX。
