# CCC 系统深度分析报告 (Round 10) — 运营·孤岛·性能·行业建议

> 仓库: hywgb/ccc · 分析基线 commit `c939518` (main, PR #1–#7 已合入，含 Round 9 修正)
> 分析时间: 2026-05-25
> 分析范围: 后端 (Go 1.25, 257 Go 文件, ~38.2K LOC) + 前端 (React 19, 66 TSX/TS 文件) + 基础设施
> 文档定位: **第十轮深度审计** — 复审 Round 9 修正后的真实质量，揭示 12 个新发现的功能孤岛 + 6 个运营流程缺口 + 7 个性能瓶颈

---

## 目录

1. [核心结论](#核心结论)
2. [Round 9 修正复审 — 「乐高积木未拼装」综合症](#round-9-修正复审--乐高积木未拼装综合症)
3. [12 个功能孤岛 (SILO-R10-1 ~ R10-12)](#12-个功能孤岛)
4. [6 个运营流程缺口 (OPS-R10-1 ~ R10-6)](#6-个运营流程缺口)
5. [7 个性能瓶颈 (PERF-R10-1 ~ R10-7)](#7-个性能瓶颈)
6. [为什么需要这些补全 — 业务影响解释](#为什么需要这些补全)
7. [行业开发建议 — 来自国内一线呼叫中心实战](#行业开发建议)
8. [Round 10 修正实施清单](#round-10-修正实施清单)

---

## 核心结论

### 表面与真实的差距

Round 9 修正合入后，PR #7 的 commit 信息称 "NATS Consumer · 审计 GET · OTEL · React Query · ESL 弹性池 · 录音加密 · bizlog" 全部修复。
然而**逐文件审计发现**：

| Round 9 声称修复项 | 真实状态 | 关键证据 |
|---|---|---|
| NATS Consumer | ❌ 仅日志，未实现真实 CDR/分析 | `postcall/worker.go` HandleMessage 只 logger.Info() |
| OTEL 追踪 | ❌ 仅 HTTP 中间件，业务无 span | `grep otel.Tracer internal/` 在 application/domain/infrastructure 0 结果 |
| React Query | ❌ Hooks 创建但**无任何页面使用** | `grep useQuery web/src/pages/` 0 结果 |
| 录音加密 | ❌ Encryptor 存在但**从未调用** | `grep RecordingEncryptor internal/` 仅自身文件 |
| bizlog | ⚠️ 仅 1 处调用 (call.ended) | `grep bizlog\\. internal/` 仅 1 个文件、1 行 |

**结论**: Round 9 完成了「积木块的生产」，但 **9 个新增基础设施中只有 2 个真正连入业务路径**（审计 GET 路径覆盖、ESL 弹性池）。其余 7 个属于「就位但未使能」的代码资产。

### Round 10 真实成熟度评估

| 维度 | Round 9 报告 | Round 10 实测 | 差异原因 |
|---|---|---|---|
| 事件驱动架构 | 45% | **20%** | NATS consumer 是空壳，后处理仍走 goroutine |
| 前端体验 | 78% (含 React Query) | **55%** | RQ 未使用，所有列表无分页/缓存 |
| 录音/合规 | 72% | **45%** | 录音文件**根本未上传 MinIO**，加密未启用 |
| 功能完整度 | 85% | **70%** | 12 个跨模块联动缺失 |
| 可观测性 | 72% | **55%** | OTEL 仅 1 层 (HTTP)，链路看不到 DB/Redis/ESL |

**调整后的整体成熟度**: 约 **62%** (相比 Round 9 自评 70% 下调 8 个百分点)

---

## Round 9 修正复审 — 「乐高积木未拼装」综合症

Round 9 的 PR #7 引入了 7 类「能力」，但绝大多数仅完成「类型/客户端创建」，未完成「业务路径接入」。这是一种典型的**「积木块综合症」**：开发以为加了文件就等于功能完成，但运行时没有任何代码路径会触发新代码。

| 能力 | 文件 | 引用计数 | 评估 |
|------|-----|---------|------|
| `postcall.Worker` | 1 文件 | 1 处订阅 | 内部仅 logger.Info，对业务无影响 |
| `crypto.RecordingEncryptor` | 1 文件 | 0 处调用 | 「死代码」 |
| `tracing.Init` | 1 文件 | 1 处启动 | 只有 HTTP 中间件 1 个 span |
| `bizlog` 全套 helpers | 1 文件 | 1 处调用 | 5 个 helper 中 4 个零调用 |
| React Query `hooks.ts` | 1 文件 | 0 处使用 | 没有一个页面切换到 hooks |
| ESL `Grow` 方法 | 已修 | 1 处调用 | 真正有用 |
| 审计 GET sensitivePaths | 已修 | 1 处调用 | 真正有用 |

**Round 10 必须完成「拼装」**，否则这些 ~640 行新增代码完全是技术债。

---

## 12 个功能孤岛

### SILO-R10-1: 数据库层面 — `calls.customer_id` 列**根本不存在**

**严重程度**: 🔴🔴 **P0** (数据丢失型缺陷)

**症状**:

```go
// internal/domain/call/entity.go:111
CustomerID *int64 `db:"customer_id" json:"customer_id,omitempty"`

// internal/domain/call/service.go:875-883
func (s *CallService) UpdateCustomerID(ctx, callID, customerID int64) error {
    c.CustomerID = &customerID
    return s.calls.Update(ctx, c)  // 调用 CallRepo.Update
}
```

但实际：

```go
// internal/infrastructure/mysql/call_repo.go:48-56
func (r *CallRepo) Update(ctx, c *call.Call) error {
    _, err := r.db.ExecContext(ctx,
        `UPDATE calls SET status=?, hangup_reason=?, disposition_code=?,
         agent_user_id=?, skill_group_id=?, hold_count=?, transfer_count=?,
         satisfaction_rating=?, ivr_duration_sec=?, ring_duration_sec=?,
         queue_duration_sec=?, wait_duration_sec=?, duration_sec=?,
         recording_url=?, answered_at=?, ended_at=? WHERE id=?`,
         /* ... 没有 customer_id ... */)
}
```

**而且 8 个 migrations 文件中没有任何一个为 `calls` 表添加 `customer_id` 列**。Round 8/9 报告中宣称 "通话↔CRM 双向关联 (CustomerID) 已修复" 是错觉。

**业务影响**:
- 客户来电 → ScreenPop 通过电话查到客户 → lifecycle 设置 `c.CustomerID = &customer.ID` → **写库时这个值被静默丢弃**
- 客户详情页查询「该客户的所有通话」永远返回空
- AI 满意度分析、客户旅程分析、客单价分析全部数据缺失

**Round 10 修正**: 添加 migration `000009_add_call_customer_id.up.sql` + 修复 `CallRepo.Create`/`Update`。

---

### SILO-R10-2: NATS postcall.Worker 是**空壳**

**严重程度**: 🔴 **P0**

`internal/application/postcall/worker.go` 的 `handleCallEnded` 只有：

```go
w.logger.Info().
    Int64("call_id", c.ID).
    Int64("tenant_id", c.TenantID).
    Msg("postcall: CDR recorded")  // 仅日志，没有任何 DB 操作或下游事件
return nil
```

与此同时 `lifecycle.EndCall` 仍然在 goroutine 中**同步串行**执行 7 件事:
1. CSAT 触发
2. Webhook 发送
3. CRM customer 记录
4. familiar agent 记录
5. dialer 释放并发
6. QA 自动质检 (interface 存在但 main.go 未 wire)
7. 通话级 AI 摘要 (从未触发)

**业务影响**: NATS event-driven 架构是装饰。一旦 `lifecycle.EndCall` panic 或耗时，所有后处理都丢。同时 NATS 流积压增长但没人真正消费。

**Round 10 修正**:
- 让 `postcall.Worker` 真正执行 CDR aggregation、AI summary trigger
- 从 lifecycle 移除可异步化的步骤（保留实时性必要的 NotifyAgent / metrics / Concurrency.Release）

---

### SILO-R10-3: QA 自动质检 — Interface 存在，**实现/接线全无**

**症状**:

```go
// internal/application/lifecycle/service.go:52-54
type QAAutoTrigger interface {
    AutoInspect(ctx context.Context, tenantID, callID int64)
}

// line 480-482  在 postCallHooksAsync 里调用
if s.qaTrigger != nil && c.AnsweredAt != nil {
    s.qaTrigger.AutoInspect(ctx, c.TenantID, c.ID)
}
```

但是：

```bash
$ grep -rn "AutoInspect" internal/
internal/application/lifecycle/service.go:54: AutoInspect(...)  # 接口定义
internal/application/lifecycle/service.go:481: ...AutoInspect(...)  # 调用
# 没有实现！
```

`QualityInspectionService` 只有 `RunInspection(ctx, tenantID, callID, schemeID, transcript)` —— 需要 scheme 和 transcript。**没有零参 AutoInspect**。

**业务影响**: 所有通话结束后的「自动质检」承诺完全是 NULL。`s.qaTrigger == nil` 始终为真。

**Round 10 修正**: 在 `QualityInspectionService` 增加 `AutoInspect(tenantID, callID)`：自动找默认 scheme + 加载 transcript + 调 RunInspection；在 main.go 中 `lifecycleSvc.SetQAAutoTrigger(qiSvc)`。

---

### SILO-R10-4: IM 自动路由 —— 函数写好但**永不触发**

**症状**:

```go
// internal/application/imrouter/service.go:46
func (s *Service) AutoRouteSession(ctx, sessionID, skillGroupID int64) error { ... }

// internal/interfaces/http/handler/widget.go:26
func (h *WidgetHandler) CreateSession(w, r) {
    sess, err := h.svc.CreateSession(...)  // 创建后停留在 waiting 状态
    response.JSON(w, http.StatusCreated, sess)
    // !!! 这里应该调用 AutoRouteSession 但没有 !!!
}
```

`IMSessionHandler.router` 字段被 main.go 通过 `SetRouter` 注入，但 `im_session.go` 全文 grep "h.router" 只有 setter，**没有任何 use site**。

**业务影响**: 网页咨询入口创建会话 → 永远停在 waiting → 坐席必须人工去 IM 列表手动 pick。IM 接入率会暴跌。

**Round 10 修正**: `WidgetHandler.CreateSession` 成功后异步 `go h.router.AutoRouteSession(ctx, sess.ID, sess.SkillGroupID)`。

---

### SILO-R10-5: 录音存储链路 —— 从未真正落 MinIO

**严重程度**: 🔴🔴 **P0** (录音是合规核心，CCC 没有录音=违法)

**全链路审计**:

```
FreeSWITCH ESL.StartRecording(channelUUID, "/recordings/{tenant}/{call}.wav")
   ↓ FreeSWITCH 把录音写到自己的本地磁盘
   ↓
lifecycle.EndCall 合成路径字符串 "/recordings/{tenant}/{call}.wav" 入库
   ↓
recordings 表写入: file_path="/recordings/X/Y.wav", status="completed"  ←  说谎
   ↓
用户调用 GET /recordings/{id}/stream
   ↓
RecordingHandler 用该 path 去 MinIO 找对象 → 404
```

**缺失环节**:
1. FreeSWITCH 录音完成 → MinIO 上传（无任何代码）
2. 上传时调用 `RecordingEncryptor.Encrypt`（Encryptor 全程零调用）
3. recordings 表的 `encryption_algo` / `encryption_key_id` 字段写入（Round 9 加了字段但 lifecycle 未写）
4. MinIO 存储后才能改 recordings.status='completed'

**Round 10 修正**: 添加 `recording.Uploader` worker —— 监听 ESL `RECORD_STOP` 事件，从 FreeSWITCH 本地拉取 → 可选加密 → 上传 MinIO → 更新 recordings.status。

---

### SILO-R10-6: TranscriptHub —— WebSocket 端点空跑

**症状**:

- 前端 `/ws/transcripts/{callID}` 端点存在
- `transcripthub.Hub.Broadcast(callID, event)` 函数存在
- 但 **grep "transcriptHub.Broadcast" 全仓 0 结果**
- ASR provider 输出从未喂给 Hub

**业务影响**: 坐席侧「实时通话转写」UI 接到 WebSocket 但永远没消息。

**Round 10 修正**: 在 `lifecycle.AnswerCall` 启动 ASR 流后，把流式回调写入 `transcriptHub.Broadcast`。

---

### SILO-R10-7: Tickets ↔ Calls 关联接口**未暴露**

**症状**:

- `TicketService.ListByCallID(ctx, callID)` 存在
- `MockTicketRepo.ListByCallID` / `mysql.TicketRepo.ListByCallID` 都实现了
- **但是 `internal/interfaces/http/router.go` 没有这条路由**

```bash
$ grep "calls/.*/tickets\|tickets/by-call" internal/interfaces/http/router.go
# 0 结果
```

**业务影响**: 坐席通话弹屏「关联工单」永远显示空白，即便工单确实关联了这通通话。

**Round 10 修正**: 添加 `GET /api/v1/calls/{id}/tickets` 路由。

---

### SILO-R10-8: Screen Pop 数据 —— 后端**返回**，前端**忽略**

**症状**:

```go
// internal/interfaces/http/handler/call_control.go:402
response.JSON(w, http.StatusOK, map[string]interface{}{
    "call":       c,
    "screen_pop": popData,   // ← 后端返回了客户信息+历史
})
```

```bash
$ grep -r "screen_pop\|screenPop\|popData" web/src
# 0 结果
```

**业务影响**: 客户来电时坐席端无法自动展示客户名称、历史交互、上次诉求 —— 「全渠道客户视图」承诺破产。

**Round 10 修正**: 前端 `phoneStore` / `IncomingCallModal` 消费 `screen_pop` 字段，在接听后渲染客户卡片。

---

### SILO-R10-9: `bizlog` 包 —— **80% 函数零调用**

**事实**:

```bash
$ grep -rn "bizlog\." internal/ | grep -v "bizlog.go"
internal/application/lifecycle/service.go:173: bizlog.CallEvent(...)   # 唯一
```

5 个 helper（Event/AgentEvent/CallEvent/IMEvent/CampaignEvent/TicketEvent）只有 1 个在 1 个地方调用。剩下的:
- 坐席状态变更（10+ 处）未用 bizlog
- IM 会话分配/转移未用 bizlog
- 工单创建/分配未用 bizlog
- Campaign 启停未用 bizlog

**业务影响**: 业务事件日志的统一索引格式无法在日志聚合（ELK/Loki）里聚合，运营无法用 `biz_event: "agent.status_changed"` 这样的查询统计某天某个租户的所有事件。

**Round 10 修正**: 在关键 application 服务方法中补 bizlog 调用。

---

### SILO-R10-10: OTEL —— 业务路径**零 span**

**症状**: `internal/interfaces/http/middleware/tracing.go` 为每个 HTTP 请求创建 1 个根 span。但是：

```bash
$ grep -rn "otel.Tracer\|tracer.Start\|trace.Span" internal/application/ internal/domain/ internal/infrastructure/
# 0 结果（除 middleware 自身）
```

**业务影响**: 一通慢通话端点延迟 800ms，但 trace 只显示 `POST /api/v1/calls/{id}/answer 800ms`，看不到 800ms 里多少在 ESL、多少在 MySQL、多少在 Redis、多少在 NATS publish。OTEL 接入毫无价值。

**Round 10 修正**: 在 lifecycle.AnswerCall / EndCall、ACD.dispatch、ESL.Originate、MySQL repo、Redis 客户端添加 span。

---

### SILO-R10-11: React Query —— **0 个页面**使用

**符号**: `package.json` 装了 `@tanstack/react-query`、`main.tsx` 配了 `QueryClientProvider`、`api/hooks.ts` 写了 10 个 hooks。**但 `web/src/pages/**/*.tsx` 没有一处 `useQuery`/`useMutation`。**

**业务影响**:
- `CrudPage` 用 `useState + useEffect + axios`，无缓存
- 切换页面再回来重新请求
- 网络抖动无自动重试
- 多页面并发请求同一资源（agents/skill-groups）会重复打后端

**Round 10 修正**: 把高流量页面（Dashboard、AgentList、SkillGroup、CallRecord）切到 hooks；为 `CrudPage` 增加 React Query 版本（不替换老接口，保留兼容）。

---

### SILO-R10-12: CampaignLiveDashboard —— 前端要 13 个字段，后端只返 4 个

**前端期望**:
```ts
interface CampaignStats {
  total_cases; completed; connected; failed; pending; in_progress;
  connect_rate; abandon_rate; avg_duration; concurrent;
  agents_active; agents_idle; elapsed_min;
}
```

**后端返回** (`DialerStats`):
```go
type DialerStats struct {
    CampaignID; ActiveCalls; TotalDialed; AbandonCount; AbandonRate; IsRunning;
}
```

**业务影响**: 监控大屏 9/13 卡片显示 0 或 "—"。运营人员看到的是「伪监控」。

**Round 10 修正**: 扩展 `DialerStats` 增加 `TotalCases/Completed/Connected/Failed/Pending/InProgress/ConnectRate/AvgDuration/Concurrent/AgentsActive/AgentsIdle/ElapsedMin`；从 `campaign_cases` 表聚合统计。

---

## 6 个运营流程缺口

### OPS-R10-1: 录音 5 步流水线断成 2 段

**应有**:
```
ESL 录音落地 → 文件可读 (5s 后) → 上传 MinIO → 加密 → DB 状态 completed
      ↑__________________________ 缺失的中间 3 步 ___________________________↑
```

**当前实现**: 只有 ESL 录音 + DB 写一个伪状态。中间 3 步全无。

**为什么必须补**: 国内监管要求录音「同步写双副本 + 7 年保留 + 可审计访问」。FreeSWITCH 本地磁盘不是合规存储；MinIO 才是。没有 Uploader，录音事实上**只存活在一个易丢的本地磁盘**。

---

### OPS-R10-2: 双轨 post-call 处理 (goroutine + NATS) 但只有 goroutine 工作

见 SILO-R10-2。

**为什么必须补**: 一旦 lifecycle 主进程崩溃，正在 goroutine 里的 post-call 任务全丢。NATS 持久化是「重启不丢」的唯一手段。

---

### OPS-R10-3: 坐席「僵尸态」清理不完整

**现状**: `ResetGhostAgents` 只扫描 `state IN ('talking','dialing')` 超过 maxDuration。
**遗漏**:
- `online`/`idle` 的僵尸（坐席客户端关了但 presence 没更新）
- `acw` 超时未恢复（应有但 acwTimers map 在进程重启后丢失）
- `break` 长时间未回归

**为什么必须补**: 真实环境中 `online` 僵尸是最常见的（关浏览器、断网）。不清理导致 ACD 错把零吞吐坐席当作可用，进入死锁。

**Round 10 修正**: ResetGhostAgents 增加 `online/idle` 超 `2 * heartbeat_interval` 自动 offline；ACW 用 Redis 而非内存 map 持久化。

---

### OPS-R10-4: 外呼 DNC 二次校验缺失

**现状**: DNC 仅在 `campaign_cases` 导入时校验一次。`dialer.eslDial` 直接拨号，不再查 DNC。
**风险**: 客户在导入后向监管/系统投诉加入 DNC，但仍持续被拨打。监管罚款 + 客户投诉。

**Round 10 修正**: `dialer.eslDial` 增加 DNC 二次确认。

---

### OPS-R10-5: 无 CDR 聚合 / 计费维度

**现状**: 通话记录在 `calls` 表逐条。**没有日/月聚合、没有按租户的费用维度统计**。

**为什么必须补**: SaaS CCC 商业模式核心是按通话分钟/接通数计费。无聚合表 → 每次出账要全表 GROUP BY。

**Round 10 修正**: 添加 `daily_cdr_summary` 表 + 凌晨 1 点聚合 worker（cron-like ticker）。

---

### OPS-R10-6: 转人工后的 IVR 上下文丢失

**现状**: IVR 收集到用户语义 `intent="账单查询"` + `slot.account_id=XXX`，但转人工时只传 skillGroupID，没把 IVR 上下文写入 call.custom_data。坐席接起后看不到「用户已经在 IVR 说了什么」。

**Round 10 修正**: `ivr/engine.go` 在转人工节点把当前 session.context 序列化到 `calls.custom_data`，前端弹屏渲染。

---

## 7 个性能瓶颈

### PERF-R10-1: Dashboard Refresher 对 1000 个租户串行刷新

**症状**:

```go
// dashboard/refresher.go:53
for _, t := range tenants {     // 1000 个租户
    r.refreshTenant(ctx, t.ID)  // 内部 6 个 SQL 查询
}
```

10s 周期 × 1000 租户 × 6 query = **6000 q/10s = 600 q/s** 仅为 dashboard 一项。

**Round 10 修正**: 并发 worker pool（10 goroutine）+ 单 SQL 多租户聚合（用 `GROUP BY tenant_id`）。

---

### PERF-R10-2: CallRepo.List 每次都全表 COUNT(*)

**症状**: `SELECT COUNT(*) FROM calls WHERE tenant_id = ?` 在 calls 表 1 亿行情况下需要扫索引 60+ 秒。

**Round 10 修正**:
- 默认列表用 cursor 分页（已实现，但 frontend 调的是 offset 版本）
- 对总数提供 `?with_total=true` 显式参数；否则不返回 total

---

### PERF-R10-3: WS Hub 写广播持有读锁迭代全量客户端

**症状**:

```go
// agenthub/hub.go (NotifyAgent)
h.mu.RLock()
for _, c := range h.clients[agentID] { ... }
h.mu.RUnlock()
```

Hub.Broadcast 之类（im/dashboard）持有 RLock 同时迭代 + 写 chan。1 个慢消费者阻塞所有人。

**Round 10 修正**: 已有 select default drop，但应进一步分租户 sharding（按 tenantID % 16 分桶减少锁竞争）。

---

### PERF-R10-4: CallRepo 全部 SELECT *

**症状**: calls 表 50+ 列，但列表场景只需 ~12 列。

**Round 10 修正**: 提供 `ListLite` 显式列，HTTP handler 默认调 ListLite。

---

### PERF-R10-5: `CrudPage` 全量拉取 + JSON.stringify 客户端搜索

**症状**: `fetchData()` 不传分页，假设后端返全部。然后:

```ts
data.filter((item) => JSON.stringify(item).toLowerCase().includes(search.toLowerCase()))
```

1 万行客户表会卡浏览器。

**Round 10 修正**: `CrudPage` 增加 `pagination` 与 `searchable.serverSide` 选项，传 `q=` 给后端。

---

### PERF-R10-6: 调用列表前端不传过滤

**症状**: `callApi.list()` 无参，后端 ListWithFilter 全空 filter → 等价于全表。

**Round 10 修正**: `CallRecordPage` 增加日期/方向/坐席筛选 UI + 传参。

---

### PERF-R10-7: ESL 连接池只 Grow 不 Shrink

**症状**: 高峰扩到 maxPool（默认 20），低谷后保持 20 连接占用 FreeSWITCH 资源。

**Round 10 修正**: 增加 `Shrink` —— 空闲 > 5 分钟回收 idle 连接到 minPool。

---

## 为什么需要这些补全

读者可能问：「Round 9 自评 70%，再做 Round 10 还有意义吗？」

**为什么 Round 9 评分虚高 8%**:
1. Round 9 审计的是「文件存在性」，没审「调用图可达性」。
2. 大量「类型/函数已声明」但没有 use site 的代码被计入「已修复」。

**为什么必须做这些细节**:
- **录音不上传 = 合规违法**（《数据安全法》《个保法》《通信短信息服务管理规定》对录音留存有强制要求）
- **客户 ID 丢失 = AI 全链路失效**（满意度预测、客户旅程、复购挖掘全部依赖 customer_id）
- **TranscriptHub 不工作 = 实时 AI 助手承诺破产**（话术提示/合规提醒/智能弹屏全失能）
- **DialerStats 缺字段 = 运营监控大屏「装样子」**
- **OTEL 仅 1 层 = 生产故障定位平均时间从 5min 退化到 1h**
- **React Query 不用 = 5000 坐席同时操作 dashboard 时打挂 API server**

每一个孤岛单独看「也能跑」，组合起来等于「客户上线后第一周就遭遇生产事故」。

---

## 行业开发建议

### 一、面向呼叫中心的「3-2-1 测试**金字塔」**

| 层 | 占比目标 | 当前 | 行动 |
|----|--------|------|-----|
| 单元测试 | 70% | ~30% (15 包) | 给 lifecycle/acd/dialer 加 mock 集成 |
| 集成测试 | 25% | 0% | docker-compose + testcontainers 启 MySQL/Redis/NATS |
| E2E (含 FreeSWITCH) | 5% | 0% | sipp 模拟主叫 + Playwright 操作坐席 UI |

### 二、运营驾驶舱必备 4 个实时看板

国内一线 CCC 团队都有但本系统全无:

1. **「红绿灯」队列健康度**：每个技能组实时 SLA 进度 (20s 内接通率 / Threshold)。**目前 Dashboard 有 service_level_20s 字段但前端 0 警戒色**。
2. **「热度图」坐席矩阵**：实时呈现每个坐席 16 小时工作热度。可用 echarts heatmap。
3. **「漏斗」客户旅程**：IVR 进入→主动转人工/等待超时分流。**目前后端有 funnel API 但前端只渲染表格**。
4. **「秒针」TPS 仪表**：API QPS、ESL 命令/s、ACD 派单/s。**目前 Prometheus 有指标但无 Grafana json 模板**。

**建议**: docs/grafana/ 目录提交 5 个 dashboard JSON。

### 三、坐席协作三件套

国内电销/客服必备：

1. **静音协助 (Coach)** — 已实现，但前端**没暴露按钮**
2. **强插 (Barge)** — 已实现，前端无入口
3. **耳语 (Whisper)** — 已实现，前端无入口
4. **会议三方 (Conference)** — 已实现，前端无入口

**建议**: 在 `MonitorPage` 增加质检员视角的 4 按钮 + WebRTC 流接入。

### 四、行业**标配但本系统缺失**的功能

| 功能 | 国内一线必备 | 本系统状态 | 建议优先级 |
|---|---|---|---|
| 智能外呼 AI 机器人 (语音对话型) | ✅ | ❌ DigitalEmployee 是文本机器人 | P1 |
| 工单自动派单 (基于 SLA 优先级) | ✅ | ⚠️ 仅手动分配 | P1 |
| 客户标签自动打标 (基于通话内容 LLM) | ✅ | ⚠️ AI 服务存在但无主动触发 | P2 |
| 录音转译多说话人区分 (说话人 diarization) | ✅ | ❌ ASR 单声道 | P2 |
| 知识库 RAG 实时辅助 (坐席端) | ✅ | ⚠️ imassist 存在但 widget 不展示 | P1 |
| 工单超时升级 | ✅ | ❌ 无 escalation 逻辑 | P1 |
| Outbound 时区合规 (本地 9:00-20:00) | ✅ | ⚠️ Round 8 OPS-5 实现但 main 未启用 | P0 |
| 多媒体 IVR (视频客服) | 中等需求 | ❌ media_type 字段已留，无引擎 | P3 |

### 五、可观测性建议三步走

1. **Phase 1 (本周内)**: OTEL 真正落入 application/infrastructure 关键点 (本 PR 提供基线)
2. **Phase 2 (1 月内)**: Prometheus 指标对接 Grafana，提供 5 个标准 dashboard JSON
3. **Phase 3 (3 月内)**: Sentry 接入错误聚合；Loki 接入业务日志（基于 bizlog 标准化）

### 六、架构演进路线

**当前 (Monolith)**:
```
[ Go single binary ] ↔ [ MySQL + Redis + NATS + MinIO + FreeSWITCH ]
```

**6 个月目标 (Modular Monolith)**:
- 按 BC 拆 `cmd/`：cmd/api（HTTP）、cmd/acd-worker（ACD goroutine 提取）、cmd/postcall-worker（NATS 消费独立部署）、cmd/dialer-worker（外呼独立）
- 这样可以单独扩 ACD/postcall，不需拆微服务

**12 个月目标 (Hybrid)**:
- ACD 抽成独立服务（FreeSWITCH 配合 inbound socket mode）
- IM 抽成独立服务（WebSocket 长连接独立 scale）
- 核心 API 保持 monolith

---

## Round 10 修正实施清单

按本 PR 实施的代码修正：

| ID | 类型 | 文件 | 行数 (估) |
|----|------|------|----------|
| FIX-1 | DB schema | `migrations/000009_round10_silo_fixes.up.sql` | +30 |
| FIX-2 | repo | `internal/infrastructure/mysql/call_repo.go` (customer_id 写入) | +15 |
| FIX-3 | postcall | `internal/application/postcall/worker.go` (真实 CDR) | +60 |
| FIX-4 | QA auto | `internal/domain/ai/service_phase9.go` (AutoInspect 方法) + main.go wire | +50 |
| FIX-5 | IM auto-route | `internal/interfaces/http/handler/widget.go` (AutoRouteSession 调用) | +15 |
| FIX-6 | Recording | `internal/application/recording/uploader.go` (新文件) + lifecycle wire | +120 |
| FIX-7 | Transcript | `internal/application/lifecycle/service.go` (TranscriptHub.Broadcast 接入) | +30 |
| FIX-8 | Tickets API | `internal/interfaces/http/router.go` + handler GET /calls/{id}/tickets | +20 |
| FIX-9 | Screen pop FE | `web/src/components/phone/*.tsx` 消费 screen_pop | +60 |
| FIX-10 | bizlog 推广 | 5 个 application 文件补 bizlog 调用 | +40 |
| FIX-11 | OTEL 业务 span | lifecycle/acd/esl/mysql 关键方法 | +80 |
| FIX-12 | React Query 接入 | DashboardPage + AgentListPage 切到 hooks | +50 |
| FIX-13 | DialerStats 扩展 | 增加 11 个字段 + campaign_cases 聚合 SQL | +80 |
| FIX-14 | Ghost agents 扩展 | online/idle 僵尸清理 | +30 |
| FIX-15 | DNC 二次校验 | dialer.eslDial 增加 | +20 |
| FIX-16 | CDR 日聚合 | `internal/application/cdr/aggregator.go` + cron | +100 |
| FIX-17 | IVR 上下文转人工 | ivr/engine 写 call.custom_data | +25 |
| FIX-18 | Refresher 并发 | dashboard/refresher.go worker pool | +30 |
| FIX-19 | CallRepo ListLite | call_repo.go 增加显式列方法 | +40 |
| FIX-20 | ESL pool Shrink | esl/client.go 增加回收 | +30 |

**预计净增**: ~900 行（含测试 ~1100 行）。涉及 25+ 文件。

---

## 结论

**Round 9 报告 70% 成熟度 → Round 10 实测 62%**，原因是 Round 9 自评把「文件已创建」算作「功能已修复」，忽略了「积木块未拼装」的硬伤。

**Round 10 必须完成的核心动作**:
1. **打通**: 12 个孤岛接线
2. **补全**: 6 个运营流程缺口
3. **优化**: 7 个性能瓶颈
4. **对齐**: 前后端字段（DialerStats、screen_pop）

**完成 Round 10 后预期成熟度: 78-82%** (真实可生产门槛 85%)

剩余到生产可用 (~85%) 的 1-2 个迭代核心: 集成测试 + Grafana dashboard + 智能外呼 AI 机器人。
