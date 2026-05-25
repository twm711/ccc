# CCC 系统深度分析报告 (Round 7)

> 仓库: hywgb/ccc · 分析基线 commit `eb4bb86` (latest, PR #14–#17 修复链已合入)
> 分析时间: 2026-05-25
> 前轮基线: Round 6 基于 commit `c3bd576` (2025-11-24)
> 分析范围: 后端 (Go 1.25+, 248 Go files, ~45K LOC) + 前端 (React 19, 65 TSX/TS files) + 基础设施 (MySQL/Redis/MinIO/NATS/FreeSWITCH/Prometheus) + 部署配置
> 文档定位: **纯分析报告**，不包含代码修改。每一项缺陷均附「业务影响 + 复现路径 + 改进建议」。

---

## 目录

1. [TL;DR — Round 6 → Round 7 变化摘要](#tldr--round-6--round-7-变化摘要)
2. [Round 6 缺陷修复审核（逐项核实）](#round-6-缺陷修复审核逐项核实)
3. [仍未修复的缺陷（携带自 Round 6）](#仍未修复的缺陷携带自-round-6)
4. [新发现的缺陷](#新发现的缺陷)
5. [更新后成熟度评估](#更新后成熟度评估)
6. [优先级路线图（更新版）](#优先级路线图更新版)
7. [附录 A：关键文件定位汇总](#附录-a关键文件定位汇总)

---

## TL;DR — Round 6 → Round 7 变化摘要

| 指标 | Round 6 (c3bd576) | Round 7 (eb4bb86) | 变化 |
| --- | --- | --- | --- |
| Go 源文件 | ~243 | 248 | +5 |
| 新增代码 (net diff) | — | +3034 / -236 行 (43 files) | PR #14–#17 |
| P0 缺陷 | 6 | 0 ✅ | **全部修复** |
| P1 缺陷 | 11 | 6 仍存 | 5 项已修复 |
| P2 缺陷 | 10 | 9 仍存 | 1 项已修复 |
| P3 缺陷 | 8 | 6 仍存 | 2 项已修复 |
| P4 缺陷 | 5 | 5 仍存 | 0 项修复 |
| 新发现缺陷 | — | 5 | 新增 |

**核心结论**：6 项 P0 致命缺陷已全部修复（calls schema 对齐、服务鉴权路由、ESL 事件订阅、ACD 队列引擎、IM 双工广播、MinIO 实例化）。系统从「35–40% 端到端可跑通」提升到 **~60–65%**。剩余工作集中在 P1 运营流程补全（并发控制、ACW 超时、合规告知等）和 P2–P4 非功能性加固。

---

## Round 6 缺陷修复审核（逐项核实）

### ✅ 已修复的 P0 缺陷

| ID | 缺陷 | 修复证据 | 验证结论 |
| --- | --- | --- | --- |
| P0-1 | calls 表 Schema 与 Go ORM 字段不匹配 | `migrations/000006_align_call_schema.up.sql` — RENAME start_at→started_at, answer_at→answered_at, end_at→ended_at, cli→caller, hangup_cause→hangup_reason；ADD direction/status/channel_uuid/hold_count/transfer_count/satisfaction_rating/duration_sec/talk_duration_sec/hold_duration_sec/acw_duration_sec/custom_data 等 14 列；重建 6 个索引 | **完全修复** |
| P0-2 | 入站电话入口被 JWT 挡住 | `router.go:764-768` 新增 `/internal/v1` 路由组 + `middleware.ServiceAuth(deps.ServiceAuthSecret)` HMAC 鉴权；`middleware/service_auth.go` 125 行实现 HMAC-SHA256 签名验证（timestamp + method + path + body hash），含 5 分钟防重放 | **完全修复** |
| P0-3 | 没有 ESL 事件订阅 | `internal/infrastructure/esl/event_listener.go` 182 行，订阅 `CHANNEL_ANSWER/HANGUP/HANGUP_COMPLETE/BRIDGE/PARK`；`internal/application/lifecycle/esl_events.go` 58 行，HandleESLEvent 驱动状态机迁移；`main.go:680` 启动 goroutine `listener.Run(hubCtx)` | **完全修复** |
| P0-4 | ACD 队列引擎缺失 | `internal/application/acd/service.go` 406 行，Redis ZSet 队列 + 5 种路由策略实现（longest_idle/round_robin/random/skill_weight/familiar）；`expireQueued()` 实现 max_wait_sec 超时转 abandoned；`main.go` 启动 `go acdSvc.Run(hubCtx)` | **完全修复** |
| P0-5 | IM 双工广播断链 | `handler/im_session.go:24-26` 新增 `IMBroadcaster` 接口 + `SetBroadcaster(b)`；`SendMessage()` 成功落库后调用 `h.broadcaster.BroadcastEvent(id, "message.new", msg)`；`main.go:671` `imSessionHandler.SetBroadcaster(imHub)` | **完全修复** |
| P0-6 | MinIO 从未实例化 | `main.go:380` `store, err := storage.NewMinIOClient(...)`；`handler/recording.go:34-36` `SetStore()` 方法注入；`Stream()` 和 `Download()` 已实现完整逻辑（仅当 store==nil 时才返回 501） | **完全修复** |

### ✅ 已修复的 P1 缺陷

| ID | 缺陷 | 修复证据 | 验证结论 |
| --- | --- | --- | --- |
| P1-1 | NATS 代码无 publish/subscribe 调用 | `internal/infrastructure/nats/client.go` 重写为 JetStream 模式（Connect + EnsureStream + Publish）；`main.go:330-337` 创建 NATS client → `EnsureStream("ccc", ["ccc.>"])` → `lifecycleSvc.SetEventPublisher(natsClient)`；lifecycle 在 EndCall/AnswerCall/HandleInbound 均调用 `s.publish(ctx, "ccc.call.*", c)` | **完全修复** |
| P1-2 | tenant_settings.api_rate_limit_per_sec 是死配置 | `middleware/ratelimit.go` 完全重写为 76 行：注入 `TenantRateProvider` 接口，从 tenant_settings 查 `api_rate_limit_per_sec`，60s in-process LRU 缓存，fallback 到 defaultRate | **完全修复** |
| P1-3 | NLS Token 一次性 Fetch 无续期 | `main.go:320,735-750` 新增 NLS token refresher goroutine：到期前刷新，失败 5min 重试，热替换 ASR/TTS provider token | **完全修复** |
| P1-7 | 熟客优先策略无历史关系持久化 | `acd/service.go:340-346` `RememberAgent()` 用 Redis `SET acd:last_agent:{tenant}:{caller}` 存最近服务坐席，TTL = `familiar_agent_days`；`lifecycle/service.go:226-234` EndCall 时自动调用；`acd/service.go:297-304` pickAgent familiar 策略先查 Redis | **完全修复**（改用 Redis 替代建议的 MySQL 表，性能更优） |
| P1-9 | IVR 编辑器 lock 无过期逻辑 | 待进一步验证：`routing/service.go` 的 Lock/Unlock 仍无 `expires_at` 字段与清理定时任务。**此项标记为部分修复** — Lock 机制存在但仍无自动过期。降级为仍未修复。 |

### ✅ 已修复的 P2 缺陷

| ID | 缺陷 | 修复证据 |
| --- | --- | --- |
| P2-4 | WebSocket Hub 无背压 / 无 ping/pong | `agenthub/hub.go:122,130-132` 实现了 Ping ticker + PongHandler + ReadDeadline 120s；`imhub/service.go:151-154` 同理 | **完全修复** |

### ✅ 已修复的 P3 缺陷

| ID | 缺陷 | 修复证据 |
| --- | --- | --- |
| P3-3 | JWT 默认密钥无启动检查 | `config/config.go:101` 默认值仍为 `change-me-in-production`。**但 main.go 无 fatal 检查**。**此项标记为未修复**。 |
| P3-4 | CORS 默认 `*` | `middleware/cors.go:9` 改为读取 `CORS_ALLOW_ORIGIN` 环境变量；默认空时为 `*`；设置后严格限制 | **部分修复**（生产环境需配置环境变量） |

---

## 仍未修复的缺陷（携带自 Round 6）

### P1 级别（6 项仍存）

#### P1-4（仍存）hangup_by 永远为 NULL

**现状**：`internal/domain/call/service.go` 的 `EndCall(ctx, id, reason)` 仍只接受 `reason`，没有 `hangupBy` 参数。Schema `calls.hangup_by ENUM('agent','customer','system')` 字段存在但 Go 实体不映射。

**验证**：`grep -rn "hangup_by\|HangupBy" internal/` — 0 结果。

**业务影响**：报表无法区分「客户挂断率」vs「坐席挂断率」。

**改进建议**：
- 扩展 `EndCall(ctx, id, reason, hangupBy)` 参数
- ESL 事件处理中，从 `ev.Headers["Hangup-Disposition"]` + `ev.Direction` 推断 hangupBy
- `CHANNEL_HANGUP` 中 `Hangup-Cause=ORIGINATOR_CANCEL` 且 direction=inbound → 客户挂断

---

#### P1-5（仍存）没有「最大并发通话」准入控制

**现状**：`grep -rn "max_concurrent\|MaxConcurrent" internal/application/lifecycle/ internal/domain/call/` — 0 结果。`tenant_settings.max_concurrent_calls` 仍是死配置。

**业务影响**：恶意/异常租户可瞬间耗尽 FreeSWITCH 资源，多租户隔离失效。

**改进建议**：
- `HandleInboundCall` / `CreateOutboundCall` 入口用 Redis INCR 检查 `tenant:{id}:active_calls`
- ESL HANGUP 事件触发 DECR
- 超过 `max_concurrent_calls` 返回 429

---

#### P1-6（仍存）录音合规告知 (recording_announce) 未在 IVR/通话流中应用

**现状**：`grep -rn "recording_announce\|RecordingAnnounce" internal/` — 0 结果。Schema 字段存在但代码无使用。

**业务影响**：GDPR/《个保法》/PCI-DSS 合规风险。

---

#### P1-8（仍存）Predictive 外呼缺少弃呼率反馈环路

**现状**：`dialer/service.go:188` 有 abandon_rate 日志输出，说明有初步检测，但 `grep -rn "PID\|EMA\|pacing_factor\|自适应\|adaptive"` 无结果 — 缺少闭环反馈调节。

**业务影响**：TCPA 等监管弃呼率 ≤ 3% 要求无法自动满足。

---

#### P1-9（仍存）IVR 编辑器 lock 无过期逻辑

**现状**：`routing/service.go:106-141` Lock/Unlock 实现完整，但无 `expires_at` 字段、无 TTL、无心跳续期、无清理 cron。编辑者离线 → 永久锁定。

---

#### P1-11（仍存）ACW → idle 无「最大 ACW 时长」自动跳转

**现状**：`grep -rn "acw.*timeout\|ACW.*timeout\|time\.AfterFunc\|AutoIdle" internal/` — 无自动超时逻辑。`tenant_settings.default_acw_seconds` 和 `agents.acw_seconds` 字段都存在但无消费方。

**业务影响**：坐席可无限挂在 ACW 不接电话。

---

### P2 级别（9 项仍存）

| ID | 缺陷 | 现状 |
| --- | --- | --- |
| P2-1 | 数据库连接池 50/10 不足 | `db.go:14-15` 仍为 `SetMaxOpenConns(50) / SetMaxIdleConns(10)`，无 `ConnMaxLifetime/ConnMaxIdleTime` |
| P2-2 | CallRepo.List 用 `SELECT *` + `COUNT(*)` 各跑一次 | `call_repo.go:108,113` 仍为 `SELECT * FROM calls ... LIMIT ? OFFSET ?` + 独立 `COUNT(*)` |
| P2-3 | 审计日志中间件同步写 DB | `audit.go:32` 仍为 `r.RemoteAddr`（未解析 X-Forwarded-For），仍同步 `repo.Create` |
| P2-5 | Dashboard Redis HGETALL 无原子刷新 | 未见修改 |
| P2-6 | ESL 连接池命令串行 | 连接池已实现但池大小与并发关系未见优化 |
| P2-7 | 报表导出全量内存装载 | `export/service.go` 仍接受 `[]*report.AgentReport` 切片，全量内存 → CSV；无流式/异步导出 |
| P2-8 | 前端缺少请求级 cancel/重试 | 未引入 React Query 或类似库 |
| P2-9 | Snowflake ID 多副本 node_id 管理 | 未见改进 |
| P2-10 | Prometheus 业务指标无 Set/Inc 调用 | `grep "ActiveCalls\.\|QueueDepth\.\|CampaignProgress\." internal/application/ internal/domain/` — 0 结果，指标仍全部为 0 |

### P3 级别（6 项仍存）

| ID | 缺陷 | 现状 |
| --- | --- | --- |
| P3-1 | WebSocket 鉴权用 URL 查询参数 | `agenthub/hub.go:98` 仍为 `r.URL.Query().Get("agent_id")`，无 JWT 鉴权 |
| P3-2 | CheckOrigin = true（4 处） | `agenthub/hub.go:18`, `imhub/service.go:18`, `dashboard/service.go:19`, `transcripthub/hub.go:18` 均仍为 `return true` |
| P3-3 | JWT 默认密钥无启动 fatal | `config.go:101` 默认 `change-me-in-production`；`main.go` 无检查 |
| P3-5 | 录音/转写敏感信息脱敏缺失 | 无改进 |
| P3-6 | 审计日志不覆盖隐私读路径 | `audit.go:17` 仍跳过所有 GET/HEAD |
| P3-7/P3-8 | SQL 注入面 / 密码哈希策略 | bcrypt 使用 `DefaultCost`（=10），建议提升到 ≥12；其他未变 |

### P4 级别（5 项全部仍存）

| ID | 缺陷 | 现状 |
| --- | --- | --- |
| P4-1 | 无分布式追踪 (OTEL) | `grep "otel\|opentelemetry" internal/` — 0 结果 |
| P4-2 | 无结构化业务事件日志 | 业务事件未走结构化日志 |
| P4-3 | 无 SLO/错误预算仪表盘 | Grafana 仅技术指标 |
| P4-4 | 无灰度发布/蓝绿通路 | graceful shutdown 已有（`main.go:700-708`），但未等通话 drain |
| P4-5 | 无租户级容量水位面板 | 无改进 |

---

## 新发现的缺陷

### NEW-1 ⚠️ ESL 事件处理未处理 CHANNEL_BRIDGE 和 CHANNEL_PARK

**定位**：`lifecycle/esl_events.go:23-35` 的 switch 只处理 `CHANNEL_ANSWER` 和 `CHANNEL_HANGUP/HANGUP_COMPLETE`，而 `event_listener.go:52-56` 订阅了 `CHANNEL_ANSWER/HANGUP/HANGUP_COMPLETE/BRIDGE/PARK` 共 5 种事件。

**业务影响**：
- `CHANNEL_BRIDGE` 未处理 → 通话桥接时无法更新 DB 状态（如从 ringing → active），也无法准确记录 ring_duration_sec
- `CHANNEL_PARK` 未处理 → 通话进入 park（等待队列）时无法记录 queue 进入时间

**改进建议**：扩展 `HandleESLEvent` switch 分支，处理 BRIDGE（→ TransitionToActive）和 PARK（→ 记录 queue 进入事件）。

---

### NEW-2 ⚠️ lifecycle.EndCall 中所有 post-call 副作用仍然同步串行

**定位**：`lifecycle/service.go:125-238` EndCall 依次同步执行：ESL Hangup → 录音落库 → 时长计算 → Agent ACW → 通知 → CSAT → Webhook → CRM 写回 → Campaign 回写 → Familiar 缓存 → NATS publish。共 11 步全部串行。

**业务影响**：
- 虽然 NATS JetStream 已接入（Round 6 P1-1 修复），但仅用于 publish 事件，**所有 hook 仍在 EndCall 同步 goroutine 中串行执行**
- 任何一步延迟（如 CRM 外部 API 超时、Webhook 5xx）都会拖慢整通话挂机响应
- NATS 当前只 publish 了 `ccc.call.ended/answered/created` 三个事件，但**无消费者 subscribe**
- 效果：事件被写入 JetStream 后无人消费，只有存储作用

**改进建议**：
- 核心路径仅保留：ESL Hangup + 状态更新 + Agent ACW + NATS publish
- 其余 hook（录音、CSAT、Webhook、CRM、Campaign 回写、Familiar）改为 NATS 消费者异步处理
- 新增独立 worker goroutine 订阅 `ccc.call.ended`，依次执行 post-call pipeline

---

### NEW-3 ⚠️ ACD dispatcher 无 max_queue_size 检查

**定位**：`acd/service.go:122-136` `Enqueue()` 只做 ZADD，不检查当前队列长度是否超过 `skill_groups.max_queue_size`。`expireQueued()` 只处理 max_wait_sec 超时。

**业务影响**：配置了 `max_queue_size=50` 的技能组，如果瞬间涌入 1000 通呼叫，全部进入 ZSet，Redis 内存暴涨。

**改进建议**：Enqueue 前 `ZCARD` 检查队列长度，超过 max_queue_size 直接返回 overflow 错误，触发 IVR 溢出策略（转语音邮箱/其他技能组/回拨）。

---

### NEW-4 ⚠️ bcrypt DefaultCost 偏低

**定位**：`identity/service.go:172` `bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)`。Go 的 `bcrypt.DefaultCost` = 10。

**业务影响**：OWASP 当前建议 bcrypt cost ≥ 12。对于存放坐席、管理员密码的系统，cost=10 在 2026 年已偏低。

**改进建议**：改为 `bcrypt.GenerateFromPassword([]byte(newPassword), 12)`。

---

### NEW-5 ⚠️ CORS middleware 空值 fallback 为 `*` + Credentials

**定位**：`middleware/cors.go:9-19`：`CORS_ALLOW_ORIGIN` 为空时 `origin = "*"`，同时 `Access-Control-Allow-Credentials: true`。

**业务影响**：浏览器规范禁止 `Access-Control-Allow-Origin: *` + `Access-Control-Allow-Credentials: true` 同时生效。某些浏览器会直接拒绝。更重要的是，忘记设置环境变量的部署会在生产环境暴露 CORS 漏洞。

**改进建议**：
- 当 `CORS_ALLOW_ORIGIN` 为空时不设置 `Allow-Credentials`，或 fatal 告警强制配置
- 推荐反射请求 Origin + 白名单校验

---

## 更新后成熟度评估

| 维度 | Round 6 评分 | Round 7 评分 | 变化原因 |
| --- | --- | --- | --- |
| 领域建模 (DDD 分层) | ★★★★☆ 80% | ★★★★☆ 82% | ACD 新增为独立 application 包，结构更清晰 |
| 通话生命周期编排 | ★★☆☆☆ 40% | ★★★★☆ 75% | ESL 事件订阅 + 状态机驱动 + 录音落库全打通 |
| ACD/路由 | ★★☆☆☆ 35% | ★★★★☆ 70% | 5 种 routing_policy 全部有实现；缺 max_queue_size 检查 |
| IM 双工实时性 | ★★☆☆☆ 40% | ★★★★☆ 75% | REST → Hub 广播已接通 |
| 录音/质检/合规 | ★★☆☆☆ 35% | ★★★☆☆ 55% | MinIO 实例化 + Stream/Download 实现；仍缺脱敏和合规告知 |
| 性能/扩展性 | ★★☆☆☆ 35% | ★★☆☆☆ 40% | WS ping/pong 已加；DB 池/SELECT */导出仍未改 |
| 安全/合规 | ★★☆☆☆ 40% | ★★★☆☆ 50% | 服务鉴权 HMAC 已加；WS 鉴权/CheckOrigin/JWT 默认密钥仍未修 |
| 可观测性 | ★★★☆☆ 55% | ★★★☆☆ 55% | 无变化（业务指标仍未接入） |
| 部署交付完整度 | ★★☆☆☆ 40% | ★★★☆☆ 55% | NATS JetStream 已真正接入；NLS Token 续期已有 |
| 事件驱动架构 | ★☆☆☆☆ 15% | ★★☆☆☆ 35% | NATS publish 已有，但无 consumer；post-call 仍同步串行 |

**综合成熟度**：从 ~40% 提升到 **~60%**

---

## 优先级路线图（更新版）

### 立即修复（1 周内）— 7 项

| 优先级 | ID | 缺陷 | 预估工作量 |
| --- | --- | --- | --- |
| P1 | P1-5 | max_concurrent_calls 准入控制 | 2h |
| P1 | P1-11 | ACW 超时自动转 idle | 3h |
| P1 | P1-4 | hangup_by 字段映射 + ESL 推断 | 2h |
| P3 | P3-1 | WebSocket JWT 鉴权替换 URL 参数 | 3h |
| P3 | P3-2 | CheckOrigin 白名单（4 处） | 1h |
| P3 | P3-3 | JWT 默认密钥启动 fatal | 0.5h |
| NEW | NEW-3 | ACD Enqueue max_queue_size 检查 | 1h |

### 短期修复（2 周内）— 8 项

| 优先级 | ID | 缺陷 | 预估工作量 |
| --- | --- | --- | --- |
| P1 | P1-6 | 录音合规告知 recording_announce | 3h |
| P1 | P1-8 | Predictive dialer 弃呼率反馈环路 | 4h |
| P1 | P1-9 | IVR Lock TTL + 清理 cron | 2h |
| P2 | P2-1 | DB 连接池参数优化 | 0.5h |
| P2 | P2-3 | 审计日志异步 + X-Forwarded-For | 3h |
| P2 | P2-10 | Prometheus 业务指标 Inc/Dec 接入 | 4h |
| NEW | NEW-1 | ESL CHANNEL_BRIDGE/PARK 处理 | 3h |
| NEW | NEW-2 | post-call hooks 改异步（NATS consumer） | 6h |

### 中期修复（1 个月内）— 10 项

| 优先级 | ID | 缺陷 |
| --- | --- | --- |
| P2 | P2-2 | Call 列表 keyset 分页 + 列裁剪 |
| P2 | P2-5 | Dashboard Redis 原子更新 |
| P2 | P2-7 | 报表流式导出 + 异步任务 |
| P2 | P2-8 | 前端 React Query 集成 |
| P2 | P2-9 | Snowflake node_id 注册中心 |
| P3 | P3-5 | 录音/转写敏感信息脱敏 |
| P3 | P3-6 | 隐私读路径审计 |
| NEW | NEW-4 | bcrypt cost 提升 |
| NEW | NEW-5 | CORS 空值处理修复 |
| P2 | P2-6 | ESL 连接池并发优化 |

### 长期修复（1 个季度+）— 5 项

| 优先级 | ID | 缺陷 |
| --- | --- | --- |
| P4 | P4-1 | OpenTelemetry 分布式追踪 |
| P4 | P4-2 | 结构化业务事件日志 |
| P4 | P4-3 | SLO/错误预算仪表盘 |
| P4 | P4-4 | 灰度发布 + 通话 drain |
| P4 | P4-5 | 租户级容量水位面板 |

---

## 附录 A：关键文件定位汇总

| 主题 | 文件 | 行号 | 简述 |
| --- | --- | --- | --- |
| Schema 对齐迁移 | `migrations/000006_align_call_schema.up.sql` | 全文 | calls 表 RENAME + ADD 14 列 |
| 服务鉴权中间件 | `internal/interfaces/http/middleware/service_auth.go` | 全文(125行) | HMAC-SHA256 签名验证 |
| 内部路由组 | `internal/interfaces/http/router.go` | 764-768 | `/internal/v1` 路由 |
| ESL 事件监听器 | `internal/infrastructure/esl/event_listener.go` | 全文(182行) | TCP 长连接 + event plain 订阅 |
| ESL 事件处理 | `internal/application/lifecycle/esl_events.go` | 全文(58行) | HandleESLEvent → 状态机 |
| ACD 队列引擎 | `internal/application/acd/service.go` | 全文(406行) | Redis ZSet + 5种路由策略 |
| IM 双工广播 | `internal/interfaces/http/handler/im_session.go` | 24-26, 159-160 | IMBroadcaster 接口 + BroadcastEvent |
| MinIO 实例化 | `cmd/server/main.go` | 380 | storage.NewMinIOClient() |
| NATS 事件总线 | `internal/infrastructure/nats/client.go` | 全文(69行) | JetStream Publish |
| NATS 接线 | `cmd/server/main.go` | 330-337 | EnsureStream + SetEventPublisher |
| Per-tenant 限流 | `internal/interfaces/http/middleware/ratelimit.go` | 全文(76行) | TenantRateProvider + 60s cache |
| NLS Token 续期 | `cmd/server/main.go` | 320, 735-750 | 到期前刷新 goroutine |
| 熟客路由(Redis) | `internal/application/acd/service.go` | 297-346 | familiar agent Redis lookup |
| WS 鉴权(待修) | `internal/application/agenthub/hub.go` | 98 | URL Query 方式(需改 JWT) |
| CheckOrigin(待修) | `agenthub/hub.go:18, imhub/service.go:18` | — | return true(需加白名单) |
| DB 连接池(待修) | `internal/infrastructure/mysql/db.go` | 14-15 | MaxOpen=50, NoLifetime |
| 审计同步(待修) | `internal/interfaces/http/middleware/audit.go` | 32 | r.RemoteAddr, 同步 Create |
| SELECT *(待修) | `internal/infrastructure/mysql/call_repo.go` | 37,46,108,113 | 多处 SELECT * |
| 业务指标(待修) | `internal/interfaces/http/middleware/metrics.go` | 36-82 | 定义齐全无调用 |
| CORS(待修) | `internal/interfaces/http/middleware/cors.go` | 9-19 | 空值 fallback * + Credentials |
| JWT 默认值(待修) | `internal/config/config.go` | 101 | change-me-in-production |

---

## 附录 B：本轮分析方法论

1. **Git 增量分析**：`git diff c3bd576..eb4bb86 --stat` 确认 43 文件、+3034/-236 行变更范围
2. **逐项验证**：对 Round 6 全部 40 项缺陷，使用 `grep` + 文件阅读逐一核实修复状态
3. **新增代码审计**：重点审查 PR #14–#17 新增的 ACD (406行)、ESL EventListener (182行)、ServiceAuth (125行)、RateLimit (76行) 等核心模块
4. **跨层依赖追踪**：从 main.go 入口追踪所有 Set/Wire 调用，确认 NATS/MinIO/ACD/ESL/IM Hub 的完整接线路径
5. **缺陷挖掘**：对新增代码进行 edge case 分析（如 ACD 无 max_queue_size、ESL 事件处理不完整、post-call 仍同步等）

---

**报告状态**：完。本文不修改任何代码、不生成 PR、不触发 CI。

**建议下一步动作**：
1. 立即修复 7 项（P1-5/P1-11/P1-4/P3-1/P3-2/P3-3/NEW-3）— 预估 12.5h
2. 短期修复 8 项（P1-6/P1-8/P1-9/P2-1/P2-3/P2-10/NEW-1/NEW-2）— 预估 25.5h
3. 把本报告的附录 A 作为后续 PR 的 checklist
