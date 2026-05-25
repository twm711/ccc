# CCC 系统深度分析报告 (Round 8) — 运营完善 · 功能孤岛打通 · 性能优化 · 行业建议

> 仓库: hywgb/ccc · 分析基线 commit `4cc451d` (含 PR #2 + PR #3 全部修复)
> 分析时间: 2026-05-25
> 前轮基线: Round 7 分析 + 修复链 (PR #1 ~ #3, 28 项缺陷已修复)
> 分析范围: 后端 Go 33,627 LOC (238 files) + 前端 React 5,297 LOC (65 files) + 基础设施 + 部署 + 迁移
> 文档定位: **运营深度分析**，聚焦流程完善、孤岛打通、性能瓶颈、行业对标建议

---

## 目录

1. [Executive Summary](#1-executive-summary)
2. [成熟度演进 — Round 6 → Round 8](#2-成熟度演进--round-6--round-8)
3. [运营流程细节缺失分析](#3-运营流程细节缺失分析)
4. [功能孤岛诊断与打通方案](#4-功能孤岛诊断与打通方案)
5. [性能瓶颈与优化路径](#5-性能瓶颈与优化路径)
6. [行业对标与开发建议](#6-行业对标与开发建议)
7. [优先级实施路线图](#7-优先级实施路线图)
8. [附录 A：缺陷清单全景](#附录-a缺陷清单全景)

---

## 1. Executive Summary

经过 Round 7 三轮修复（PR #1 文档 → PR #2 19 项修复 → PR #3 9 项修复），CCC 系统核心功能链（来电 → IVR → ACD → 坐席应答 → 通话 → 挂机 → 后处理）已基本可运行。系统成熟度从 Round 6 的 ~40% 提升至 **~75%**。

**但在"可运行"与"可运营"之间仍存在显著差距。** 本轮分析从四个维度审视：

| 维度 | 现状 | 目标 | Gap |
|------|------|------|-----|
| 运营流程完整度 | 核心通话链完整，但缺少排班/坐席复位/数据归档/合规审计 | 每日可交付班次运营 | 12 项缺失 |
| 功能模块孤岛度 | 语音/IM/AI/CRM/工单各自成线，联动断裂 | 全渠道统一路由+统一上下文 | 8 个断裂点 |
| 性能可扩展性 | 单实例可运行，但多实例/高并发路径存在瓶颈 | 1000 坐席/10,000 并发 | 6 项瓶颈 |
| 行业合规+最佳实践 | 基本安全框架已就位，但缺工信部合规/录音生命周期管理/灾备 | 等保二级+行业标杆 | 10 项差距 |

---

## 2. 成熟度演进 — Round 6 → Round 8

| 子系统 | Round 6 | Round 7 修复后 | Round 8 现状评估 | 差距说明 |
|--------|---------|---------------|----------------|---------|
| **呼入链路 (IVR→ACD→应答)** | 30% | 80% | 80% | 需补：排队超时溢出、多级IVR菜单计时统计 |
| **呼出链路 (外呼/Campaign)** | 40% | 75% | 75% | 需补：号码池轮转疲劳度管控、外呼时间窗合规 |
| **坐席状态管理** | 50% | 85% | 85% | 需补：签入/签出审计、自动归位、跨天清零 |
| **IM 全渠道** | 35% | 50% | 50% | 需补：IM路由引擎远弱于语音ACD、无排队等待 |
| **AI 引擎** | 45% | 55% | 55% | 需补：QA自动触发、数字员工与IVR集成、训练闭环 |
| **录音与合规** | 25% | 65% | 65% | 需补：录音存储分级、过期自动清理、密钥管理 |
| **报表与导出** | 60% | 80% | 80% | 需补：实时仪表板数据源、导出格式可配置 |
| **工单与CRM** | 50% | 55% | 55% | 需补：通话-工单自动关联、客户360度视图 |
| **部署与运维** | 40% | 65% | 65% | 需补：配置热加载、优雅关机完善、健康检查深度化 |
| **安全与合规** | 35% | 70% | 70% | 需补：操作审计全面化、数据脱敏规则可配置 |

**综合成熟度：~75%** (从 Round 6 的 ~40% → Round 7 的 ~60% → Round 8 的 ~75%)

---

## 3. 运营流程细节缺失分析

### 3.1 坐席日常运营流程断裂

#### OPS-1：签入/签出无审计，跨天状态不清零 [P1]

**现状**：`identity.AgentPresenceService.CheckIn/CheckOut` 更新状态但不记录签入时段。如果坐席连续在线跨越 0 点，第二天报表中的在线时长计算将跨天累加，导致数据失真。

**业务影响**：
- 坐席明明昨天 23:50 签入、今天 00:10 签出，报表上显示今天在线 20 分钟但实际工作 0 分钟
- 班次统计不准确，无法计算出勤率、迟到率

**建议**：
1. 增加 `agent_shift_log` 表 (`agent_id`, `shift_date`, `check_in_at`, `check_out_at`, `total_online_sec`)
2. 在 0 点触发跨天切割：自动结束当天班次、开始新班次
3. 报表按 `shift_date` 聚合而非 `created_at` 范围查询

#### OPS-2：坐席状态自动归位缺失 [P2]

**现状**：当 FreeSWITCH 连接中断时（网络闪断、FS 重启），坐席可能停留在 `talking` 或 `dialing` 状态永不恢复。系统没有心跳检测机制来发现"幽灵通话"。

**业务影响**：幽灵坐席占用 ACD 派发名额，其他排队中的呼叫无法被分配。

**建议**：
1. 后台 goroutine 定期（每 60s）扫描 `talking` 状态超过 `max_call_duration`（如 4h）的坐席，自动 reset 为 `idle`
2. ESL 事件监听断线时，对当前活跃通话批量触发 `EndCall(SYSTEM_ERROR)`

#### OPS-3：ACD 排队溢出路由未闭环 [P2]

**现状**：`SkillGroup.OverflowGroup` 字段已定义 (`overflow_group_id`)，但 `acd/service.go` 的 `dispatchOne` 中从未读取该字段。当一个技能组队列满时，呼叫直接被拒绝（`QueueRejected`），不会溢出到备用技能组。

**业务影响**：忙时客户被直接挂断而非转到其他可用坐席组。

**建议**：
```
// acd/service.go: Enqueue()
if qLen >= sg.MaxQueueSize {
    if sg.OverflowGroup != nil {
        return s.Enqueue(ctx, callID, *sg.OverflowGroup, priority)
    }
    // reject only if no overflow target
}
```

#### OPS-4：排班/班次管理缺失 [P2]

**现状**：系统有 `business_hours` 表管理工作时间，但无坐席排班功能。运营管理者无法预排班次、无法设置轮休、无法查看排班覆盖率。

**业务影响**：500+ 坐席的呼叫中心必须依赖外部排班工具，无法在 CCC 内闭环管理。

**建议**：
1. 新增 `schedule`、`shift_template`、`agent_schedule` 域模型
2. 集成 `business_hours` 以自动检查"当前是否在排班时段内"
3. 排班覆盖率预警：预测未来 7 天各时段人力是否充足

#### OPS-5：Campaign 外呼时间窗无合规校验 [P2]

**现状**：`dialer/service.go:isWithinSchedule()` 只检查星期几，不检查小时段。中国工信部要求外呼时间为工作日 09:00-20:00，但系统可在凌晨 3 点拨出。

**业务影响**：违规外呼可导致投诉、罚款、号码被运营商封禁。

**建议**：
```go
func (s *Service) isWithinSchedule(c *campaign.Campaign) bool {
    // ... existing weekday check ...
    hour := now.Hour()
    if hour < 9 || hour >= 20 {
        return false
    }
    return true
}
```
并在 Campaign 实体增加 `schedule_start_hour`、`schedule_end_hour` 可配置字段。

#### OPS-6：录音生命周期管理缺失 [P2]

**现状**：录音创建后永久保留在 MinIO，无自动分级（hot → warm → cold）和过期清理。`TenantSettings.RecordingRetentionDays` 已定义但从未读取。

**业务影响**：
- 存储成本线性增长
- GDPR/个保法要求数据不得超期保留

**建议**：
1. 后台 cron job 每日扫描 `recordings WHERE created_at < now - retention_days`
2. 到期后移到 cold storage 或直接删除（根据策略）
3. `TenantSettings.RecordingStorageBackend` 用于指定 hot/cold bucket

### 3.2 通话流程细节缺失

#### OPS-7：通话保持/恢复无时长统计 [P3]

**现状**：`call.HoldCount` 字段在 entity 中定义，但 `CallService` 中的 `HoldCall`/`RetrieveCall` 方法只更新状态为 `held/active`，不记录每次保持的开始/结束时间。

**业务影响**：报表中 `avg_hold_duration_sec` 永远为 0，无法评估坐席服务质量。

**建议**：在 `call_events` 中记录 `hold_start` / `hold_end` 事件，在 `CalculateDurations` 中累加。

#### OPS-8：转接/三方通话无完整事件链 [P3]

**现状**：`call.CallType` 支持 TRANSFER/CONFERENCE 等类型，但 `lifecycle.Service` 没有 `TransferCall`、`Conference` 等编排方法。坐席在前端点击转接时，底层只有 ESL 命令发出，CCC 系统的通话状态未跟随变化。

**业务影响**：转接后通话记录中看不到转接链路（A→B→C），报表中的 `transferred_calls` 统计不准确。

#### OPS-9：Dashboard 数据源为 Redis 快照，非实时聚合 [P3]

**现状**：`DashboardOverview` 通过 Redis HGETALL 读取预计算快照，但没有后台 job 定期刷新这个快照。实际上，这个快照是在 `UpdateOverview` 被调用时写入的，但 **从未有代码调用 UpdateOverview**。

**业务影响**：Dashboard 页面始终显示空数据或过期数据。

**建议**：
1. 新增后台 goroutine（每 5-10s）聚合当前活跃通话、坐席状态、队列深度
2. 或改为前端直接查 MySQL 聚合（适合小规模部署）

#### OPS-10：CSAT 满意度调查无触发条件配置 [P3]

**现状**：`csat.Service.TriggerSurvey` 被无条件调用（每通来电结束后），但 `CSATConfig` 表有 `enabled` 字段却未被检查。

**业务影响**：所有通话都触发调查，客户体验差。应支持按比例抽样、按通话时长过滤、按技能组过滤。

#### OPS-11：Callback 回呼无排程优先级 [P3]

**现状**：`callback.Scheduler.ProcessAllPending` 按 DB 查询顺序（`ListAllPending`）处理所有 pending callback，没有优先级排序（如 VIP 客户优先）。

**建议**：增加 `priority` 字段，按 priority DESC + scheduled_at ASC 排序。

#### OPS-12：数据归档/清理策略缺失 [P4]

**现状**：`call_events`、`audit_logs`、`agent_presence_log` 等高频写入表没有自动归档/清理。虽然 `deployment.md` 提到了分区策略，但没有自动化执行。

**建议**：
1. 实现 partition rotation cron job
2. 建议集成到应用层的 scheduled tasks 中

---

## 4. 功能孤岛诊断与打通方案

### 整体诊断

CCC 系统的功能模块呈现**竖井式开发**：语音(call/lifecycle)、IM(imhub/im)、AI(advancedai/qa)、CRM(crm/customer)、工单(ticket) 各自有独立的数据模型和服务层，但跨模块联动断裂。

```
┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│  Voice   │  │    IM    │  │    AI    │  │   CRM    │  │  Ticket  │
│ (calls)  │  │(sessions)│  │ (qa/de) │  │(customer)│  │ (ticket) │
└────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘
     │             │             │              │              │
     │    ❌ 无联动  │   ❌ 无联动  │    ❌ 无联动   │    ❌ 无联动  │
     └─────────────┴─────────────┴──────────────┴──────────────┘
```

### SILO-1：语音通话 ↔ IM 会话 —— 全渠道路由断裂 [P1]

**现状**：
- 语音通话通过 `acd/service.go` 的 Redis ZSET 队列 + 5 种路由策略进行分配
- IM 会话通过 `imrouter/service.go` 的 `RouteSession()` 直接指定坐席，**无排队、无路由策略、无技能组匹配**
- 两个渠道的坐席可用性各自独立判断

**业务影响**：
- 客户从 IM 转到电话时，丢失上下文（聊天记录不可见）
- IM 坐席不检查 `max_chat_slots`，可能过载
- 同一客户在 IM 和电话中被分配到不同坐席

**建议**：
1. **统一路由层**：抽象 `UnifiedRouter` 接口，语音和 IM 共用路由策略
2. **IM 排队机制**：IM 会话也进入 Redis ZSET 队列，复用 ACD 逻辑
3. **会话上下文传递**：`call.Call` 增加 `related_im_session_id`，升级时自动关联
4. **坐席并发槽位管理**：语音占 1 个 slot，IM 占按 `max_chat_slots` 配额

### SILO-2：通话记录 ↔ CRM 客户 —— 联系历史碎片化 [P2]

**现状**：
- `lifecycle.postCallHooksAsync` 中通过 `customerSvc.FindByPhone` + `RecordInteraction` 建立通话与客户的关联
- 但这个关联是**单向的**：CRM 知道有一通电话，但通话详情页不展示客户信息
- 来电 Screen Pop 只在接听瞬间触发一次，坐席刷新页面后数据消失

**业务影响**：坐席无法查看客户的完整联系历史，每次接听都像"第一次"。

**建议**：
1. `Call` entity 增加 `customer_id` 外键，在通话创建时即时绑定
2. 通话详情 API 返回客户信息 + 最近 N 次交互记录
3. Screen Pop 数据持久化到通话记录而非仅推送 WebSocket

### SILO-3：QA 质检 ↔ 通话录音 —— 自动质检未打通 [P2]

**现状**：
- QA 系统有完整的规则引擎（`qaRuleRepo`, `qaSchemeRepo`, `qaResultRepo`）+ LLM 集成
- 但质检**必须手动触发**（通过 `POST /qa/inspect` API 传入 transcript）
- 录音→转写→质检链路没有自动化串联

**业务影响**：海量录音堆积，质检覆盖率极低（行业标准：≥5% 自动质检覆盖）。

**建议**：
1. 在 `postCallHooksAsync` 中增加自动质检触发：
   - 录音完成 → ASR 转写 → 匹配 QA scheme → RunInspection
2. 支持按比例抽样（如 10% 的通话自动质检）
3. 质检结果关联到通话记录和坐席绩效

### SILO-4：数字员工(DE) ↔ IVR 引擎 —— AI 对话断裂 [P2]

**现状**：
- IVR 引擎有 `NodeDigitalEmployee` 节点类型，已注册 handler
- 但 `DigitalEmployeeHandler` 的实现（`nodes_interaction.go`）是 stub：只返回 `"default"` exit，不调用实际 DE 服务
- Advanced AI 的 `HandleConversationTurn` 具有完整的对话管理能力，但没有被 IVR 调用

**业务影响**：IVR 流程中的"智能机器人"节点实际上是透传，无法进行多轮对话。

**建议**：
1. `DigitalEmployeeHandler.Handle()` 调用 `advancedai.HandleConversationTurn`
2. 当 DE 建议转人工时，走 IVR 的 `transfer` exit → ACD 队列

### SILO-5：工单 ↔ 通话 —— 服务闭环断裂 [P3]

**现状**：
- 工单系统独立运行（CRUD + 模板 + 评论）
- 通话结束后不自动创建工单
- 坐席在通话中无法快速创建关联工单

**业务影响**：客户问题需要跨通话跟进时，坐席必须手动创建工单并手动填写关联信息。

**建议**：
1. 通话详情页增加"创建工单"按钮，自动预填通话信息
2. 工单 API 增加 `call_id` 关联字段
3. 配置化：按通话类型/持续时长自动创建工单

### SILO-6：知识库 ↔ 坐席辅助 —— 智能推荐断裂 [P3]

**现状**：
- 知识库有分类 + 文章的完整 CRUD
- IM Assist (`imassist/service.go`) 使用 LLM 生成回复建议
- 但 IM Assist **不检索知识库内容**，纯靠 LLM 模型的通用知识

**业务影响**：坐席辅助建议与企业实际话术/产品信息脱节。

**建议**：
1. IM Assist 在调用 LLM 前，先检索知识库中的相关文章作为 context
2. 实现 RAG (Retrieval-Augmented Generation) 模式
3. 对话中识别关键词自动弹出相关知识条目

### SILO-7：绩效考核 ↔ 报表数据 —— 指标未闭环 [P3]

**现状**：
- `PerformanceScorecard` 域模型已定义（`ai/service_phase9.go`）
- 但 scorecard 的数据来源是手动填写，不自动从报表聚合
- 报表(`AgentReport`)中有 `Utilization`、`AnswerRate`、`ServiceLevel20s` 等 KPI，但这些不会自动写入绩效卡

**建议**：
1. 周期性（每日/每周）自动聚合报表数据 → 更新绩效卡
2. 绩效卡增加与 KPI 目标的对比（达标/未达标）

### SILO-8：社交渠道 ↔ IM 核心 —— 消息格式不统一 [P4]

**现状**：
- WeChat/Weibo 社交渠道有独立的 API adapter（`wechat/api.go`, `weibo/api.go`）
- `SocialChannelService` 独立于核心 `IMService` 运行
- 社交渠道的消息不经过统一的 IM 路由和排队

**建议**：社交渠道消息统一注入 IMService 的消息管道，复用路由和分配逻辑。

---

## 5. 性能瓶颈与优化路径

### PERF-1：ACD 轮询全员状态 — O(n) per tick [P1]

**现状**：`acd/service.go:pickAgent()` 每次派发都遍历技能组全部成员（`GetBySkillGroup` 拉全表），再逐一查询 `GetByAgentID` 获取在线状态。

**复杂度**：假设 100 个技能组，每组 50 人，500ms 一轮 → 每秒 10,000 次 DB/Redis 查询。

**建议**：
1. **坐席状态 Redis 化**：`HSET agent:presence:{agent_id} status idle`，直接 HSCAN 代替逐一查询
2. **空闲坐席集合**：维护 `SET sg:{id}:idle_agents`，`pickAgent` 从集合中 SRANDMEMBER/SPOP
3. 复杂度从 O(members) 降至 O(1)

### PERF-2：DashboardHub 广播序列化瓶颈 [P2]

**现状**：`dashboard/service.go:BroadcastAll()` 在单个 goroutine 中遍历所有客户端，为每个 tenant_id 查询并缓存 overview。当 200 个管理员同时在线，每次广播需要串行处理 200 个客户端。

**建议**：
1. 改为 fan-out：per-tenant 维护客户端列表，overview 计算一次后广播给同 tenant 的所有客户端
2. 当前已有 `tenantData` 缓存机制，但外层 for-range 仍是串行的

### PERF-3：报表查询无 DB 层聚合 [P2]

**现状**：`report.ReportService` 的 `ListAgentReports` 等方法直接返回 `[]*AgentReport`，没有 SQL 层 GROUP BY 聚合。具体聚合逻辑不在代码中可见，疑似依赖前端聚合或 MySQL 视图。

**建议**：
1. 确保报表查询使用 MySQL GROUP BY + 索引覆盖
2. 高频查询（日报/周报）结果预计算到汇总表，避免每次全表扫描
3. 考虑 ClickHouse 或 TiDB 作为分析层

### PERF-4：Webhook 无速率控制 [P2]

**现状**：`webhook.Deliver()` 为每个匹配的 config 启动一个 goroutine 发起 HTTP 请求（最多 3 次重试，每次退避 2-6 秒）。高峰期一个 tenant 有 10 个 webhook config，每通电话产生 3 个事件 → 30 个并发 goroutine。

**建议**：
1. 使用 worker pool（bounded goroutines）替代无限制 `go`
2. 每个 webhook config 增加 rate limit 配置
3. 失败积压超过阈值时自动禁用并告警

### PERF-5：ESL 连接池缺少健康检查 [P3]

**现状**：`esl.Client` 使用固定大小连接池（`PoolSize` 配置），但没有连接存活检测。FreeSWITCH 重启后，池中连接全部失效，命令开始报错但不重连。

**建议**：
1. 连接池增加 ping/keepalive 检测
2. 命令执行失败时尝试重连并重试一次
3. 连接池满时排队而非立即报错

### PERF-6：MySQL 大表缺少读写分离 [P3]

**现状**：所有读写共享一个 `*sql.DB` 连接。报表查询（可能涉及百万行扫描）与在线事务共用同一连接池。

**建议**：
1. 读写分离：引入 `ReadDB` + `WriteDB`，报表/导出/审计读取走只读实例
2. 连接池已配置 `ConnMaxLifetime(30m)` + `ConnMaxIdleTime(5m)`，这部分正确

---

## 6. 行业对标与开发建议

### 6.1 对标分析

| 维度 | CCC 当前 | 行业标杆 (Genesys/Avaya/阿里云CCC) | 差距 |
|------|---------|------|------|
| **全渠道统一路由** | 语音ACD完整，IM无排队 | 语音+IM+邮件+社交统一队列，统一SLA | ★★★ 高 |
| **智能质检** | LLM已集成，手动触发 | 100%录音自动质检，实时情绪预警 | ★★☆ 中 |
| **坐席辅助** | IM Assist有基础LLM，无RAG | 知识库RAG + 实时话术推荐 + 下一步建议 | ★★☆ 中 |
| **预测式外呼** | 4种模式已实现，弃呼率节流完善 | + 预测接通率模型 + 空闲坐席预测 | ★☆☆ 低 |
| **实时监控** | Prometheus指标丰富，Dashboard快照 | 实时大屏 + 异常告警 + 自动扩缩容 | ★★☆ 中 |
| **合规安全** | JWT+CORS+PII脱敏已就位 | + 等保二级 + 录音加密 + 操作水印 | ★★☆ 中 |
| **多租户隔离** | 逻辑隔离（tenant_id过滤） | 物理隔离可选 + 资源配额 + 计费 | ★★☆ 中 |
| **灾备恢复** | 无HA、无备份策略 | 跨AZ主备 + RPO<1min + 自动故障转移 | ★★★ 高 |

### 6.2 行业开发建议

#### REC-1：建立统一会话模型 (Unified Interaction) [战略级]

**为什么需要**：当前语音 `Call`、IM `IMSession`、工单 `Ticket` 是三个完全独立的实体。行业标杆产品都使用**统一交互模型**：

```go
type Interaction struct {
    ID            int64
    TenantID      int64
    CustomerID    *int64
    Channel       string   // voice | webchat | wechat | email
    SkillGroupID  int64
    AgentUserID   *int64
    Status        string
    ParentID      *int64   // 用于升级/转接链路
    Metadata      json.RawMessage
    StartedAt     time.Time
    EndedAt       *time.Time
}
```

**收益**：
- 路由层只需一套 ACD 逻辑
- 报表自然跨渠道聚合
- 客户 360° 视图天然完整

**实施路径**：渐进式——不重写现有模型，而是在上层增加 `Interaction` 聚合层。

#### REC-2：实现 NATS Consumer 事件驱动架构 [高优先]

**为什么需要**：当前 NATS JetStream 只有 Publisher（lifecycle 发布 `ccc.call.*` 事件），**没有 Consumer**。这意味着：
- 事件发出后没人消费
- QA 自动质检、CRM 联动、报表实时更新、告警触发都必须同步耦合在 lifecycle 中

**建议架构**：
```
lifecycle.EndCall()
    ↓ publish "ccc.call.ended"
    
consumer-qa:     subscribe "ccc.call.ended" → auto QA inspect
consumer-report: subscribe "ccc.call.*"     → update dashboard
consumer-alert:  subscribe "ccc.call.*"     → check SLA breach
consumer-crm:    subscribe "ccc.call.ended" → update interaction
```

**收益**：解耦、可横向扩展、新功能只需增加 consumer 而不修改核心代码。

#### REC-3：WebRTC 直连替代 SIP 话机 [中期规划]

**为什么需要**：当前坐席必须有 SIP 话机或 SIP 软电话才能通话。行业趋势是浏览器内直接 WebRTC 通话：
- 前端已有 `AgentPhoneBar.tsx` 组件
- FreeSWITCH 已配置 mod_verto (端口 8081/8082)
- 但前端没有 WebRTC 客户端实现

**建议**：
1. 集成 SIP.js 或 Opal WebRTC SDK 到 AgentPhoneBar
2. 通过 mod_verto WSS 连接 FreeSWITCH
3. 渐进式替换：WebRTC 与 SIP 并存

#### REC-4：录音与通话数据加密静态存储 [合规必须]

**为什么需要**：
- 中国《个人信息保护法》要求通话录音需加密存储
- 当前录音以明文 WAV 存储在 MinIO
- 录音文件名包含 tenant_id 和 call_id（可推断业务关系）

**建议**：
1. MinIO 启用 SSE-S3 或 SSE-KMS 服务端加密
2. 录音文件名改用随机 UUID（去关联化）
3. 下载/流式播放时通过 presigned URL + 有效期控制

#### REC-5：引入配置热加载 [运维改善]

**为什么需要**：当前所有配置通过环境变量在启动时加载（`config.Load()`），变更任何配置必须重启服务。对于 500+ 坐席的生产系统，重启意味着所有通话中断。

**建议**：
1. 运行时可变配置（如限流阈值、ACD 参数、LLM model 切换）存入 Redis/DB
2. 通过 admin API 或 tenant settings 页面调整
3. 坐席数量、技能组配置等已在 DB 中，无需变更

#### REC-6：实现优雅关机完善 [P2]

**为什么需要**：当前 `main.go` 的 shutdown 逻辑只给了 10s 超时。在此期间：
- WebSocket 连接被粗暴断开
- 正在进行的 post-call hooks 可能被中断
- ACD 队列中的等待呼叫不会被转移
- `/readyz` endpoint 已就位但没有在 shutdown 时调用 `SetReady(false)`

**建议**：
```go
// signal received
handler.SetReady(false)     // 立即停止接收新流量
hubCancel()                 // 停止后台任务
time.Sleep(5 * time.Second) // drain 期：让 LB 感知不可用
srv.Shutdown(ctx)           // 等待在途请求完成
```

#### REC-7：前端状态管理升级 [P3]

**为什么需要**：当前前端使用基础的 `auth.ts` store，没有全局状态管理。WebSocket 推送的实时数据（坐席状态变更、通话事件、IM 消息）缺少统一的状态订阅机制。

**建议**：引入 Zustand 或 Redux Toolkit + RTK Query，统一管理 WebSocket 事件流。

#### REC-8：测试覆盖率提升 [P3]

**为什么需要**：
- 现有测试主要覆盖 domain 层 (call, routing, identity 等)，共 ~15 个 test 文件
- application 层几乎无测试（lifecycle, acd, dialer 等核心逻辑无测试）
- 接口层无集成测试
- 前端无测试

**建议**：
1. 优先为 `lifecycle.Service` 和 `acd.Service` 编写单元测试（mock repo）
2. 集成测试：使用 testcontainers-go 启动 MySQL + Redis，端到端验证通话链路
3. 前端：至少为 AgentPhoneBar 和 IVR Editor 写 E2E 测试

#### REC-9：API 版本管理 [P4]

**为什么需要**：当前所有 API 都在 `/api/v1` 下，没有版本演进策略。一旦需要 Breaking Change，所有客户端（前端、Webhook、第三方集成）都需要同时升级。

**建议**：
1. 保持 `/api/v1` 稳定，新功能/破坏性变更放 `/api/v2`
2. 使用 Content-Type versioning 或 URL versioning
3. 旧版本 sunset 时给 6 个月过渡期

#### REC-10：可观测性三支柱补齐 [P2]

**为什么需要**：
- **Metrics (指标)**：✅ 已完善 — Prometheus 业务指标丰富
- **Logging (日志)**：✅ 基本完善 — zerolog 结构化日志 + request_id
- **Tracing (链路追踪)**：❌ 完全缺失 — 无 OpenTelemetry/Jaeger 集成

一通电话经过 IVR → ACD → 坐席应答 → 转接 → 挂机，涉及 5+ 服务调用，没有 trace_id 无法排查延迟来源。

**建议**：
1. 引入 OpenTelemetry Go SDK
2. HTTP middleware 注入 trace context
3. ESL 命令、Redis 操作、DB 查询自动上报 span
4. 导出到 Jaeger 或 Tempo

---

## 7. 优先级实施路线图

### Phase 1 — 运营可用（2-3 周）

| # | 项目 | 来源 | 预估工作量 |
|---|------|------|-----------|
| 1 | Dashboard 数据源实时化 | OPS-9 | 2d |
| 2 | ACD 溢出路由闭环 | OPS-3 | 1d |
| 3 | 坐席状态自动归位 | OPS-2 | 1d |
| 4 | 签入/签出审计+跨天切割 | OPS-1 | 2d |
| 5 | 外呼时间窗合规 | OPS-5 | 0.5d |
| 6 | 优雅关机完善 | REC-6 | 0.5d |

### Phase 2 — 孤岛打通（3-4 周）

| # | 项目 | 来源 | 预估工作量 |
|---|------|------|-----------|
| 7 | IM 统一排队路由 | SILO-1 | 5d |
| 8 | 通话-CRM 双向关联 | SILO-2 | 2d |
| 9 | QA 自动质检触发 | SILO-3 | 3d |
| 10 | 数字员工 ↔ IVR 打通 | SILO-4 | 2d |
| 11 | 工单-通话关联 | SILO-5 | 2d |
| 12 | 知识库 RAG 集成 | SILO-6 | 3d |

### Phase 3 — 性能与合规（2-3 周）

| # | 项目 | 来源 | 预估工作量 |
|---|------|------|-----------|
| 13 | ACD 坐席状态 Redis 化 | PERF-1 | 3d |
| 14 | Webhook worker pool | PERF-4 | 1d |
| 15 | 录音存储加密+生命周期 | OPS-6/REC-4 | 3d |
| 16 | OpenTelemetry 集成 | REC-10 | 3d |
| 17 | NATS Consumer 架构 | REC-2 | 5d |

### Phase 4 — 差异化竞争力（4-6 周）

| # | 项目 | 来源 | 预估工作量 |
|---|------|------|-----------|
| 18 | 统一交互模型 | REC-1 | 10d |
| 19 | WebRTC 浏览器通话 | REC-3 | 8d |
| 20 | 排班管理 | OPS-4 | 5d |
| 21 | 实时情绪预警 | 行业趋势 | 3d |
| 22 | 测试覆盖率提升 | REC-8 | 持续 |

---

## 附录 A：缺陷清单全景

### 历史修复统计

| 轮次 | 新增 | 修复 | 累计修复 | 累计剩余 |
|------|------|------|---------|---------|
| Round 6 | 40 | 0 | 0 | 40 |
| Round 7 分析 | 5 | 11 (R6修复) | 11 | 34 |
| PR #2 | 0 | 19 | 30 | 15 |
| PR #3 | 0 | 9 | 39 | 6 |
| **Round 8 分析** | **36** | — | 39 | **42** (含新增) |

### Round 8 新增缺陷分类

| 优先级 | 数量 | 类别 |
|--------|------|------|
| P1 | 3 | OPS-1, SILO-1, PERF-1 |
| P2 | 14 | OPS-2~6, SILO-2~4, PERF-2~4, REC-6, REC-10 |
| P3 | 12 | OPS-7~11, SILO-5~7, PERF-5~6, REC-7~8 |
| P4 | 7 | OPS-12, SILO-8, REC-1~5, REC-9 |

### 文件定位索引

| 组件 | 关键文件 | 行数 |
|------|---------|------|
| 主入口 | `cmd/server/main.go` | 786 |
| 通话生命周期 | `internal/application/lifecycle/service.go` | 468 |
| ACD 引擎 | `internal/application/acd/service.go` | 419 |
| 外呼引擎 | `internal/application/dialer/service.go` | 345 |
| 通话域 | `internal/domain/call/service.go` | 901 |
| IVR 引擎 | `internal/application/ivr/engine.go` | 188 |
| IM 域 | `internal/domain/im/service.go` | 249 |
| IM 路由 | `internal/application/imrouter/service.go` | 41 |
| Dashboard Hub | `internal/application/dashboard/service.go` | 153 |
| Webhook | `internal/application/webhook/service.go` | 137 |
| 配置 | `internal/config/config.go` | 173 |
| 路由表 | `internal/interfaces/http/router.go` | ~800 |
| 前端入口 | `web/src/App.tsx` | 129 |
| 部署文档 | `docs/deployment.md` | 252 |

---

> 本报告为纯分析文档，不包含代码变更。建议按 Phase 1~4 路线图逐步实施。
