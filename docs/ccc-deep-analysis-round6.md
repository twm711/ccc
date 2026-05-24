# CCC 系统深度分析报告 (Round 6)

> 仓库: twm711/ccc · 分析基线 commit `c3bd576` (latest, 11-PR 修复链已合入)
> 分析时间: 2025-11-24
> 分析范围: 后端 (Go 1.26, ~40K LOC) + 前端 (React 19) + 基础设施 (MySQL/Redis/MinIO/NATS/FreeSWITCH/Prometheus) + 部署配置
> 文档定位: **纯分析报告**，不包含代码修改。每一项缺陷均附「业务影响 + 复现路径 + 改进建议」。

---

## 目录

1. [TL;DR — 当前成熟度评估](#tldr--当前成熟度评估)
2. [P0 致命缺陷（阻塞业务运行）](#p0-致命缺陷阻塞业务运行)
3. [P1 运营流程缺口（功能上线即遇到）](#p1-运营流程缺口功能上线即遇到)
4. [P2 性能与可扩展性瓶颈](#p2-性能与可扩展性瓶颈)
5. [P3 安全与合规缺口](#p3-安全与合规缺口)
6. [P4 可观测性与运维缺口](#p4-可观测性与运维缺口)
7. [行业对标建议（Genesys / Avaya / Five9 / AWS Connect）](#行业对标建议)
8. [优先级路线图](#优先级路线图)

---

## TL;DR — 当前成熟度评估

| 维度 | 评分 | 关键问题 |
| --- | --- | --- |
| 领域建模 (DDD 分层) | ★★★★☆ 80% | 分层清晰；少数边界服务（lifecycle）依然是“胶水代码 + REST 编排”形态 |
| 通话生命周期编排 | ★★☆☆☆ 40% | **ESL 事件订阅缺失**、**ACD 队列引擎缺失**、Inbound 入口 **JWT 阻塞** |
| ACD/路由 | ★★☆☆☆ 35% | skill_groups 配置完整但 **routing_policy 未被任何代码消费**，依赖 mod_callcenter 却未提供其 conf |
| IM 双工实时性 | ★★☆☆☆ 40% | REST 发消息时 **未广播给 IMHub WebSocket**，访客侧无实时性 |
| 录音/质检/合规 | ★★☆☆☆ 35% | MinIO 客户端就绪但 **从未在 main.go 中实例化**；播放/下载返回 501 |
| 性能/扩展性 | ★★☆☆☆ 35% | 连接池 50/10、N+1 查询、`SELECT *`、审计同步落库、广播无背压 |
| 安全/合规 | ★★☆☆☆ 40% | WS 鉴权用 URL 查询参数、`CheckOrigin=true`、JWT 默认密钥未强校验、CORS `*` |
| 可观测性 | ★★★☆☆ 55% | Prometheus 指标定义齐全但 **大多数业务指标无 .Set/.Inc 调用** |
| 部署交付完整度 | ★★☆☆☆ 40% | docker-compose 列出 NATS/Kafka/MinIO，**代码全未接入**；FreeSWITCH 缺 dialplan/directory/callcenter.conf |

**核心结论**：从代码量与领域覆盖看像 70%+ 的产品，但 **几条主链路存在断点**（schema 与 ORM 字段不对、ACD 引擎缺失、ESL 事件未订阅、Inbound 入口无服务鉴权），实际可跑通的端到端业务流大概只有 35–40%。这一轮分析的目标是把这些“形似而不通”的地方一次性挑出来。

---

## P0 致命缺陷（阻塞业务运行）

### P0-1 ❗ `calls` 表 Schema 与 Go ORM 字段全面不匹配

**定位**：
- 迁移脚本 `migrations/000001_init_schema.up.sql:387-435` 定义 `calls` 表，列为：`start_at / answer_at / end_at / cli / call_type / hangup_cause`，**没有** `direction / status / started_at / answered_at / ended_at / hangup_reason / disposition_code / phone_number_id / carrier_id / sip_trunk_id / hold_count / transfer_count / satisfaction_rating / recording_url / custom_data / channel_uuid` 等列。
- Go 实体 `internal/domain/call/entity.go:83-119` 与 Repo `internal/infrastructure/mysql/call_repo.go:19-32` **全部使用** 上述「缺失」列名 INSERT/UPDATE。

**业务影响**：
- 任何 `POST /calls/inbound`、`POST /calls/dial`、外呼活动派发，**第一条 INSERT 就会** `Error 1054: Unknown column 'direction' in 'field list'`。
- `go build` 通过、单测通过（因为单测全部走 mock repo），但**对接真实 MySQL 即崩**。这是产品上线前必须解决的「假成熟」问题。

**复现路径**：
1. `make docker-up` 启动完整环境
2. `curl -X POST localhost:8080/api/v1/calls/inbound -H "Authorization: Bearer ..."` 立即返回 500
3. MySQL 日志：`Unknown column 'direction' in 'field list'`

**为什么会出现**：领域模型与迁移脚本由不同阶段产出，缺少 **schema → 代码生成** 或 **集成测试串联**。Mock repo 单测无法发现这类不对齐。

**改进建议**：
- 短期：把 Go 端 `Call` 实体的 `db:""` tag 全量对齐到 `start_at/answer_at/end_at/cli/call_type/hangup_cause`，并在 calls 表上 `ALTER` 补齐 `status / hangup_reason / phone_number_id / carrier_id / sip_trunk_id / hold_count / transfer_count / satisfaction_rating / recording_url / custom_data / channel_uuid / direction / disposition_code` 等业务字段（要么改 schema 适配代码，要么改代码适配 schema）。
- 中期：引入 `sqlboiler / sqlc / xo` 之类 **从 schema 生成 ORM struct**，让 Go 端只保留 DTO/聚合根，DB schema 作为唯一事实来源。
- 长期：CI 流水线增加「跑迁移 → 跑集成测试」的真实数据库阶段。

---

### P0-2 ❗ 入站电话入口被 JWT 中间件挡住，FreeSWITCH 无法触达

**定位**：
- `internal/interfaces/http/router.go:177-181`：`/api/v1` 整个路由组挂载 `middleware.Auth(JWT)` + `RateLimit` + `AuditLog`。
- `internal/interfaces/http/router.go:389`：`POST /calls/inbound` 在此组内。
- 没有任何 **服务到服务（FreeSWITCH→Go）** 的鉴权机制（HMAC、mTLS、内部 service-account JWT、IP 白名单等）。

**业务影响**：
- 真实呼入流程理应是：电信线路 → SIP/SS7 → FreeSWITCH 收到 INVITE → FreeSWITCH dialplan/Lua 脚本 HTTP `POST /calls/inbound` 通知 Go 创建通话记录并启动 IVR。FreeSWITCH 没有 JWT，无法穿透 Auth 中间件。
- 当前实现里 inbound 端点必须由前端调用 —— 这违反 CCC 系统的事件驱动本质（前端不可能感知一通真实来电）。

**复现路径**：
1. FreeSWITCH 收到 SIP INVITE
2. dialplan 中执行 `<action application="curl" data="http://api:8080/api/v1/calls/inbound ..."/>`
3. Go 返回 `401 missing or invalid authorization header`
4. 通话无对应业务记录，IVR/录音/路由全断

**为什么会出现**：路由分层只考虑「面向用户」的 BFF 风格鉴权，没区分「公网用户」与「内网设备」。

**改进建议**：
- 引入 **服务鉴权专用中间件**：基于 `X-CCC-Service-Token`（HMAC-SHA256，时间戳 + 防重放）或 mTLS。
- 单独挂出 `/internal/v1/calls/...` 路由组，使用上述中间件。
- 推荐设计：FreeSWITCH 侧使用 `mod_xml_curl` 让 Go 提供动态 dialplan，则只需 IP 白名单即可。

---

### P0-3 ❗ 没有 ESL 事件订阅 —— 整个通话状态机退化为「人工驱动」

**定位**：
- `internal/infrastructure/esl/client.go` 仅有 **outbound 命令** (`Originate / Hold / Bridge / Eavesdrop` 等)，没有任何 `events all / events plain CHANNEL_*` 订阅。
- `grep -rn "OnEvent\|HandleEvent\|CHANNEL_HANGUP\|CHANNEL_ANSWER\|CHANNEL_BRIDGE" internal/` 返回 0 行业务代码。

**业务影响**：
- 通话状态变迁完全靠**前端 / 接口调用方主动 POST**：`/calls/{id}/answer`、`/calls/{id}/end` 等。
- 真实场景下：
  - 客户挂断 → FreeSWITCH 发 `CHANNEL_HANGUP` → **Go 无人监听** → DB 中 `calls.status` 永远停在 `active`
  - `duration_sec / ring_duration_sec / queue_duration_sec` 全部依赖 Go 内部时钟，与 FreeSWITCH 实际事件错位
  - 录音停止、CSAT 触发、CRM 写回等所有「post-call hook」永远不会触发

**复现路径**：
1. 外呼通话建立后，客户挂断
2. FreeSWITCH 输出 `2024-xx-xx CHANNEL_HANGUP_COMPLETE`
3. Go DB 中该 `call.id` 仍为 `status=active, ended_at=NULL`
4. Dashboard `active_calls` 计数永不下降

**改进建议**：
- 新增 `internal/infrastructure/esl/event_listener.go`：常驻 goroutine，连接 ESL `event plain CHANNEL_PARK/ANSWER/BRIDGE/HANGUP/PROGRESS_MEDIA`。
- 事件入 NATS / channel，由 `lifecycle.Service` 消费驱动状态机迁移与 post-call hook。
- 同步幂等：以 `channel_uuid` 为主键去重，避免重复回调。

---

### P0-4 ❗ ACD 队列引擎缺失 —— `routing_policy` 是死配置

**定位**：
- `migrations/000001_init_schema.up.sql:109-121` 定义 `skill_groups.routing_policy ENUM('longest_idle','least_utilized','round_robin','skill_level','familiar_agent')` 与 `max_queue_size / max_wait_sec / overflow_target`。
- `internal/domain/identity/entity.go:88-97` Go 侧定义了 `RoutingPolicyRoundRobin/LeastRecent/Random/SkillWeight/Familiar`。
- 但 **没有任何代码** 读取这些字段去做 **「队列里的下一通电话该派给谁」** 的决策。
- `internal/application/lifecycle/service.go` 的 `TransitionCallToRinging` 需要外部传入 `agentUserID`，自身不做选人。
- `internal/application/ivr/nodes_routing.go:42` 把队列任务直接抛给 FreeSWITCH 的 `callcenter:%s@default` —— 但 `deploy/freeswitch/conf/autoload_configs/` **没有 callcenter.conf.xml**，mod_callcenter 也不会自动加载。

**业务影响**：
- 配置了 5 种路由策略，实际生效 0 种。
- 「最大队列长度」「最长等待时间」「溢出策略 (voicemail/transfer/callback/reject)」全部不生效，超载时无降级。
- 「最熟坐席 (familiar_agent)」直接相关客户满意度与单解率指标，缺失会显著影响 NPS。
- 高峰时段会出现「通话进入 queue 状态后永远不出队」的卡死。

**复现路径**：
1. 创建 skill_group，配置 `routing_policy=longest_idle, max_wait_sec=60, overflow_target=voicemail`
2. 10 通呼入打进同一 skill group
3. Go 侧 calls 表 10 条记录全部 `status=queue` 永久停留
4. 60s 后没有自动转语音邮箱

**为什么会出现**：架构层把 ACD 决策「外包」给 FreeSWITCH mod_callcenter，但没有同时提供 callcenter.conf、没有把 skill_groups 表同步到 mod_callcenter 的 queues 配置、Go 侧也未感知 mod_callcenter 事件。两边都假设对方会处理。

**改进建议**：
- 推荐方案 A（自研 ACD）：在 `internal/application/acd/` 新建队列引擎：
  - **Redis Sorted Set** 存等待队列（key=`queue:{skill_group_id}`, score=`enqueue_ts + priority_weight`）。
  - **Lua 脚本 + Pub/Sub** 实现 “取队首 + 选 agent + 原子分派”。
  - 选 agent 时按 routing_policy 查 `agent_presence` 表与 `idle_since` 字段。
  - 定时任务扫超时（>max_wait_sec）转 overflow_target。
- 推荐方案 B（继续走 mod_callcenter）：
  - 在 `deploy/freeswitch/conf/autoload_configs/callcenter.conf.xml` 配置 queue & strategy。
  - 在 Go 启动时 push skill_groups 配置到 FS（`callcenter_config queue load`）。
  - 订阅 `CC_*` 事件（`AGENT_OFFERING`, `MEMBER_QUEUE_START`, `MEMBER_QUEUE_END`）回写 calls 表。

---

### P0-5 ❗ IM 双工实时性断链：REST 发消息 → WebSocket 不广播

**定位**：
- `internal/interfaces/http/handler/im_session.go:132-150` `SendMessage` 调用 `h.svc.SendMessage`，直接返回 201。
- 没有调用 `imhub.Hub.Broadcast` 把消息推送给已连接的 WS 客户端。
- `internal/application/imhub/service.go:78-92` Broadcast 只被 WebSocket inbound 消息触发。

**业务影响**：
- 坐席从 PC 发消息走 REST → 访客（浏览器/小程序）走 WS → **访客侧永远收不到坐席消息**，除非主动刷新会话。
- 同样反向：访客走 webchat REST 发消息，坐席侧 WS 也收不到。
- IM 渠道实际是「半双工 + 轮询」的客户体验，不能算 omni-channel。

**复现路径**：
1. 访客通过 WS 连入 `wss://.../api/v1/ws/im?session_id=123`
2. 坐席调用 `POST /api/v1/im/sessions/123/messages`
3. WS 端没有任何消息推送

**改进建议**：
- 在 `IMSessionHandler` 注入 `*imhub.Hub`，`SendMessage` 成功落库后调用 `h.hub.Broadcast(sessionID, IMEvent{Type:"message", ...})`。
- 长期：把 IM 消息发送统一收敛到 `imhub.Service`，由它负责「持久化 + 广播 + webhook 触发 + AI hooks」。

---

### P0-6 ❗ MinIO 客户端定义齐全但**从未实例化**，录音播放/下载全部 501

**定位**：
- `internal/infrastructure/storage/minio.go` 实现了 `MinIOClient.Upload/Download/GetPresignedURL`。
- `cmd/server/main.go` 全文 grep `MinIOClient / NewMinIOClient` 无任何引用。
- `internal/interfaces/http/handler/recording.go:49-57` `Stream` 与 `Download` 直接返回 `501 NotImplemented`。

**业务影响**：
- 前端 `web/src/pages/call-records/` 中的录音播放功能 100% 不可用。
- QA 质检无法听录音、无法转写、AI 摘要走不通（依赖录音文件的链路全断）。
- 录音保留期（`tenant_settings.recording_retention_days`）配置无对应清理任务。

**改进建议**：
- 在 `cmd/server/main.go` 加 `storage.NewMinIOClient(...)` → `EnsureBucket` → 注入 `RecordingHandler`。
- 实现 `Stream`：用 `Range` header + presigned URL 重定向 / 反向代理。
- 实现 FreeSWITCH 录音完成后的 `RECORD_STOP` 事件钩子 → 上传 MinIO + 写 recordings 表 + 释放本地文件。
- 实现 retention 定时任务（cron job）扫描超期录音执行 DELETE。

---

## P1 运营流程缺口（功能上线即遇到）

### P1-1 NATS 与 Kafka 在 docker-compose 中存在，但代码**完全没有 publish/subscribe 调用**

**定位**：`internal/infrastructure/nats/client.go` 仅定义 Connect/Publish，但 `grep "natsClient.Publish\|nats.Publish"` 在 application/domain 层 0 结果。

**业务影响**：
- 「事件驱动」是 CCC 的核心架构假设之一（CDR 推 BI、IM 推 webhook、AI 异步打分等都依赖事件总线）。
- 现在所有 hook 都是 **同步串行 in-process 调用**（lifecycle.EndCall 里串行写录音/CSAT/webhook/CRM/campaign）。任何一步抖动会拖慢整通话挂机。

**改进建议**：
- 抽象 `domain/event.Publisher` 接口（Publish(ctx, topic, payload)）。
- 在 EndCall 等聚合根操作后，发出 `call.ended.v1` 等领域事件。
- 由独立 worker（call-postprocess、qa-runner、cdr-exporter）订阅。
- 关键链路（计费、合规）使用 NATS JetStream / Kafka 的 **at-least-once** 语义 + 幂等消费。

---

### P1-2 `tenant_settings.api_rate_limit_per_sec` 是死配置

**定位**：`internal/interfaces/http/middleware/ratelimit.go:10` `RateLimit(limiter, defaultRate int)`，在 `router.go:179` 写死 `100`。

**业务影响**：多租户场景下「VIP 租户买 1000 QPS、低价租户限 50 QPS」的差异化能力不存在；运营无法限流投诉来源租户。

**改进建议**：中间件改造为读取 `tenant_settings.api_rate_limit_per_sec`（建议 LRU 缓存 30s 避免每个请求查 DB）。

---

### P1-3 NLS Token 一次性 Fetch，**没有续期任务**

**定位**：`cmd/server/main.go:258-267` 启动时调用 `FetchNLSToken`，结果存 `nlsToken` 字符串，注入 ASR/TTS Provider。

**业务影响**：阿里云 NLS Token TTL 默认 ~24h，第二天进程不重启就会 ASR/TTS 全部静默失败（Provider 看不到 401 重试触发点）。

**改进建议**：
- Token 管理改为 `TokenManager` goroutine：到期前 N 分钟主动续期；获取失败做指数退避并打 Prometheus 指标 `ccc_nls_token_refresh_failures_total`。
- Provider 内部支持「Token Refresh callback」热替换，避免重启。
- 同时支持 `STS Token + AccessKey` 双模式，云原生 KMS 集成。

---

### P1-4 通话「客户主动挂断」事件无法识别 → `hangup_by` 永远为 NULL

**定位**：`internal/domain/call/service.go:127` `EndCall(ctx, id, reason)` 只接收 `reason`，没有 `hangup_by`。Schema `calls.hangup_by ENUM('agent','customer','system')` 字段定义存在但 Go 实体不映射。

**业务影响**：
- 报表无法区分「客户挂断率」「坐席挂断率」（KPI 必看项）。
- 合规：根据 `tenant_settings.hangup_policy='agent_only'` 时如果客户先挂应触发警报，目前无法触发。

**改进建议**：扩展 `EndCall(ctx, id, reason, hangupBy)`，配合 P0-3 的 ESL 事件订阅，从 `Hangup-Disposition` 头自动判定。

---

### P1-5 没有「最大并发通话」与「全局并发熔断」

**定位**：`tenant_settings.max_concurrent_calls = 100`，但 `CreateOutboundCall / HandleInboundCall` 入口均不检查当前租户并发数。

**业务影响**：
- 恶意/异常租户可以瞬间把 FreeSWITCH 拉爆（线路、CPU、带宽）影响其他租户。
- 多租户隔离失效。

**改进建议**：
- Redis INCR/DECR 维护 `tenant:{id}:active_calls`，超过 `max_concurrent_calls` 直接 429。
- ESL HANGUP 事件触发 DECR。

---

### P1-6 录音「客户合规告知」(`recording_announce`) 配置存在但**未在 IVR/通话流中应用**

**定位**：`tenant_settings.recording_announce BOOLEAN DEFAULT FALSE` 在 schema 中存在；`grep -rn "recording_announce" internal/` 在 application 层无任何使用。

**业务影响**：GDPR/《个保法》/PCI-DSS 等合规场景必须告知客户「您的通话将被录音」。配置缺失 → 罚款 / 业务暂停。

**改进建议**：IVR 引擎接入 Welcome 节点前自动 prepend `recording_announce` 音频；并写入 ivr_tracking 节点变量留证。

---

### P1-7 「熟客优先 (familiar_agent)」策略需要的「历史坐席关系表」未持久化

**定位**：`tenant_settings.familiar_agent_days = 30` 配置存在；但「客户最近 30 天内被哪位坐席服务过」需要回溯 calls 表 JOIN customers，**无现成索引**。

**业务影响**：每次入队时回溯 30 天 JOIN，几千万行 calls 表会拖死 MySQL。

**改进建议**：
- 新增 `customer_agent_relations` 表（customer_id, agent_user_id, last_served_at, count, primary KEY (customer_id, agent_user_id)）。
- EndCall 时 `INSERT ... ON DUPLICATE KEY UPDATE last_served_at`。
- ACD 选人时直接 `SELECT agent_user_id FROM customer_agent_relations WHERE customer_id=? AND last_served_at > NOW() - INTERVAL ? DAY ORDER BY last_served_at DESC LIMIT 5`。

---

### P1-8 外呼活动「预测式 (Predictive)」缺少弃呼率反馈环路

**定位**：`internal/application/dialer/service.go` predictive 模式以固定参数算 pacing；schema 中虽有 `campaign.abandon_rate` 字段但 dialer 启动后无周期性回写。

**业务影响**：
- TCPA 等监管要求「弃呼率 ≤ 3%」，超标会被运营商封号 / 巨额罚款。
- 没有反馈环路 → 永远跑同一组参数 → 难以稳定通过监管。

**改进建议**：
- 每 30s 计算窗口期内 `(abandoned/total_dialed)` 写入 `campaigns.abandon_rate`；超阈值自动收紧 `pacing_factor`。
- 引入 PID 控制器或简单 EMA + 阈值控制。

---

### P1-9 IVR 编辑器 lock/unlock 没有「锁过期」逻辑

**定位**：`web/src/api/endpoints.ts:40-41` 暴露 lock/unlock；服务端 `ivr_flow_service` 锁的所有权与 TTL 未在 schema/service 中体现。

**业务影响**：编辑者突然离线（关浏览器、网络断）会 **永久锁定 flow**，其他人无法编辑必须 admin 介入。

**改进建议**：lock 表加 `expires_at`；前端心跳每 30s 续期；服务端 `cron job` 清理超期锁。

---

### P1-10 短信通知 / 邮件通知 / Webhook 重试机制薄弱

**定位**：`internal/application/webhook/service.go` 与 SMS handler；grep 未发现 backoff / DLQ / 去重表。

**业务影响**：Webhook 一次 5xx 即丢消息；客户 OA/CRM 集成方常常抱怨「漏单」。

**改进建议**：所有出站集成走 outbox 模式 + 独立 worker 重试 + DLQ + 告警。

---

### P1-11 坐席状态机：`acw` → `idle` 没有「最大 ACW 时长」自动跳转

**定位**：`tenant_settings.default_acw_seconds = 30` 配置存在；前端 `AgentPhoneBar.tsx:133` 手动点「完成」回 idle；**没有任何后台定时器** 强制超时跳转。

**业务影响**：坐席可以一直挂在 ACW 不接电话（混日子 / 故意挂机），导致排队等待延长。该指标是国际 CCC 的硬性 KPI。

**改进建议**：进入 ACW 时启动 `time.AfterFunc(acw_seconds)` 自动转 idle 并推送 agent-events；前端 `AgentPhoneBar` 显示倒计时。

---

## P2 性能与可扩展性瓶颈

### P2-1 数据库连接池 50/10 不足以支撑设计目标

**定位**：`internal/infrastructure/mysql/db.go:14-15` `SetMaxOpenConns(50) / SetMaxIdleConns(10)`。

**问题**：
- 单进程 50 个 MySQL 连接 → 100 个并发坐席每秒做 2 次写 + 报表 + 仪表盘轮询，瞬间打满。
- 没有 `SetConnMaxLifetime`，长连接遇到 MySQL `wait_timeout`（默认 8h）后会偶发 `invalid connection`。
- 没有 `SetConnMaxIdleTime`。

**建议**：
```go
db.SetMaxOpenConns(200)        // 与目标 QPS / RT 联立
db.SetMaxIdleConns(50)
db.SetConnMaxLifetime(30*time.Minute)
db.SetConnMaxIdleTime(5*time.Minute)
```
- 进一步：业务读多写少，引入 **读写分离 + sqlx.DB[primary] + sqlx.DB[replica]**。

---

### P2-2 `CallRepo.List` 用 `SELECT *` + 多条件动态 WHERE + `COUNT(*)` 各跑一次

**定位**：`internal/infrastructure/mysql/call_repo.go:61-105`。

**问题**：
- `SELECT *` 在 calls 表（数十列、含 JSON 与 TEXT 字段）扫描成本高。
- `COUNT(*)` 与查询 SQL 分两次走 DB（无法走 covering index）。
- WHERE 中 `caller LIKE '%xxx%'` 前导通配符 → **全表扫描**。
- `started_at` 索引上文已证不存在该列名（schema 是 `start_at`）。

**建议**：
- 列出真正需要的 20 个列（含 `id, call_type, cli, callee, status, start_at, end_at, agent_user_id, skill_group_id, duration_sec, recording_url`）；JSON/TEXT 字段延迟到详情页加载。
- 用 `keyset pagination`：`WHERE start_at < ? ORDER BY start_at DESC LIMIT ?`，避免 `OFFSET` 深翻页 O(n)。
- 模糊查询用 ngram 索引或 ES（呼叫记录通常需要全文检索 transcript + summary，ES 是行业标配）。
- COUNT 查询独立缓存（5 秒粒度 ok）。

---

### P2-3 审计日志中间件**同步写 DB**

**定位**：`internal/interfaces/http/middleware/audit.go:26-37` 在 handler 返回后同步 `repo.Create`。

**问题**：
- 每个 POST/PUT/DELETE 多一次 DB round-trip（10–30ms）。
- DB 抖动会拖慢业务请求 P99。
- `r.RemoteAddr` 不解析 X-Forwarded-For，反代后所有审计 IP 都是网关。

**建议**：
- 改异步：channel 缓冲 + 批量插入；channel 满时降级丢弃并打 metric。
- 解析 `X-Forwarded-For / X-Real-IP`，配合可信代理列表。
- 大流量场景把审计写到 ClickHouse / OpenSearch。

---

### P2-4 WebSocket Hub 广播无背压、Send chan 满直接丢消息

**定位**：
- `internal/application/agenthub/hub.go:86-90` 与 `imhub/service.go:88-92` 均使用 `select { case c.Send <- data: default: }` —— **客户端慢则消息静默丢**。
- 没有 ping/pong / read deadline / 慢消费者主动断连。

**问题**：
- 客户端弱网时丢失关键事件（call_ringing/call_answered）→ 前端状态机错乱。
- 长时间 idle 的客户端无法被识别为「僵尸」。

**建议**：
- 慢消费者：3 次发送失败 → 主动 close conn → 客户端重连即可重置。
- 加 `conn.SetReadDeadline + Pong handler`，30s 内未收到 Pong 主动断开。
- 关键事件（呼入振铃）改为 **WS + REST 兜底**（前端 onClose 时 fallback 轮询 `/api/v1/agents/me/active-calls`）。

---

### P2-5 Dashboard 仪表盘指标走 Redis HGETALL，无原子刷新

**定位**：`internal/infrastructure/redis/dashboard.go:27-58`。

**问题**：
- HGETALL 一次取 17 个 field 没问题；但更新 (UpdateOverview) 与读 (GetOverview) 之间无原子保护 → 高并发下 Dashboard 短时出现「半新半旧」混合。
- 没有 metric TTL：租户停用后 key 留在 Redis 内存中永远。

**建议**：
- 写入用 Pipeline 一次发送，或使用 RedisJSON。
- key 上加 `EXPIRE 86400`，每次 update 时 refresh。

---

### P2-6 ESL Client 连接池只有 1 个 Persistent Connection（依据 PR #11 修复历史推断），命令串行

**定位**：`internal/infrastructure/esl/client.go` 实现了连接池 + 断路器，但并发坐席同时发 `uuid_hold`、`uuid_transfer` 时仍可能在单 conn 上排队。

**建议**：
- 池大小绑定 `min(N_AGENT/10, 32)`。
- 关键命令（Hangup/HoldRetrieve）允许打到不同连接并发执行。
- 命令延迟纳入 `ccc_esl_command_duration_seconds` 直方图 P99 监控。

---

### P2-7 报表导出全量内存装载

**定位**：`internal/application/export/service.go`。

**问题**：导出 CSV/Excel 时一次性 `SELECT * FROM calls WHERE ... ORDER BY start_at` 装载到内存。10 万行起 OOM。

**建议**：
- 改 `rows.Next()` 流式 → `csv.Writer.Write` → 直接 `http.Flusher`。
- 大于一定行数走异步导出（生成 job_id + 邮件/通知投递下载链接）。

---

### P2-8 前端 endpoint.ts 缺乏请求级别 cancel/重试/去抖

**定位**：`web/src/api/client.ts` 27 行，应该是裸 axios。

**问题**：
- 仪表盘页 useEffect 切换时旧请求未 cancel → 顺序错乱。
- 任何 5xx 直接抛错 → 重试需用户手动。

**建议**：
- 接入 React Query（@tanstack/react-query）：自动缓存 + 重试 + cancel + retry-on-window-focus。
- 错误页/重试按钮统一组件化。

---

### P2-9 Snowflake ID 与高并发写入

**定位**：`pkg/snowflake/snowflake.go`（未读取，但被全量 ID 生成调用）。

**问题**：单进程多 goroutine 调用 `snowflake.NextID()` 会出现毫秒内冲突（取决于实现）；多副本部署需要每个 `SNOWFLAKE_NODE_ID` 唯一，没有看到注册中心管理。

**建议**：使用 etcd / 数据库 lease 分配 node_id；或迁移到 ULID（去中心、无碰撞前提下无需 node_id 管理）。

---

### P2-10 Prometheus 业务指标定义齐全但**几乎不被 Set/Inc**

**定位**：`internal/interfaces/http/middleware/metrics.go` 定义了 `ActiveCalls / ActiveAgents / ESLConnections / QueueDepth / CampaignProgress`，但 `grep "ActiveCalls\."` / `QueueDepth\.` 在业务代码中均无结果。

**问题**：仪表盘看到的 `ccc_active_calls` 一直是 0。

**建议**：在 lifecycle / ACD / dialer 各状态转换处 Inc/Dec 这些 gauge；引入 `metrics.go` 统一封装。

---

## P3 安全与合规缺口

### P3-1 ❗ WebSocket 鉴权完全依赖 URL 查询参数

**定位**：`internal/application/agenthub/hub.go:96-100` `r.URL.Query().Get("agent_id"/"tenant_id")`。

**问题**：
- URL 不走 Auth 中间件（WS 路由挂在 `/api/v1` 之外）。
- 任何人改 query 串可冒充任意 agent_id 接收事件 → 偷听呼叫元数据 + DDOS。
- URL 参数会出现在 access log / 代理 log / 浏览器历史中。

**建议**：
- WS 握手时通过 `Sec-WebSocket-Protocol: bearer.<jwt>` 或第一帧消息发 JWT 鉴权。
- 校验 JWT 中 user_id == 请求的 agent_id，tenant 一致。
- 失败立刻关连。

---

### P3-2 `CheckOrigin = true`（允许任意 Origin）

**定位**：`agenthub/hub.go:18 / imhub/service.go:18 / dashboard/transcripthub` 全部 `CheckOrigin: func(r *http.Request) bool { return true }`。

**问题**：CSRF 攻击者诱导坐席浏览器访问恶意页面 → 用 WS 偷取实时事件。

**建议**：白名单 `Origin` 校验，从环境变量 `CORS_ALLOW_ORIGIN` 复用。

---

### P3-3 JWT 默认密钥未做生产环境强制校验

**定位**：`docker-compose.yml:18` `JWT_SECRET=${JWT_SECRET:-change-me-in-production}`，main.go 无启动时 `if secret == "change-me-..." { fatal }`。

**问题**：忘改密钥的部署会签发可被预测的 Token。

**建议**：main.go 启动校验：
- 默认占位符 → fatal
- 长度 < 32 → warn
- 推荐 RS256 + KMS 托管私钥

---

### P3-4 CORS 默认 `*` 且 Allow-Credentials 未严格控制

**定位**：`middleware/cors.go:11`。

**问题**：开发环境默认 `*` 可接受；生产部署忘改即放任所有源。

**建议**：默认拒绝 + 显式白名单；OPTIONS 与正常请求共享同一 origin 检查。

---

### P3-5 录音/转写/IM 消息**敏感信息脱敏**机制缺失

**定位**：schema 仅有 `masked_callee` 字段；缺银行卡 / 身份证 / 手机号 / 地址 等正则脱敏管线。

**问题**：
- PCI-DSS 要求录音不留 PAN；目前录音原文持久化无任何脱敏。
- 转写文本写 transcripts 表，AI 摘要写 calls.ai_summary → 监管审计高风险。

**建议**：
- 引入「质检前」管线：ASR 文本 → 正则 / NER 脱敏 → 落库。
- 录音里的 DTMF 录制段（信用卡输入）使用 FreeSWITCH `playback_terminators + uuid_record_pause`。

---

### P3-6 审计日志读路径未覆盖（GET 全部跳过）

**定位**：`middleware/audit.go:16-17` 跳过所有 GET/HEAD。

**问题**：「查看通话录音 / 客户资料 / 报表」属于隐私读路径，监管（GDPR/HIPAA/《数据安全法》）要求审计。

**建议**：白名单关键 GET（`/recordings/*/stream`、`/customers/{id}`、`/reports/*/export`）写审计。

---

### P3-7 SQL 注入面：`call_repo.List` 用 `where + args` 拼字符串

代码用了 `?` 占位符，没有字符串拼接到 SQL —— 这一点 OK。但其他 repo (`grep` 检查) 仍需逐一确认；建议引入 sqlx safe build helper / squirrel。

---

### P3-8 密码 / API Key 哈希策略

**定位**：`migrations/000005_add_password_hash.up.sql`。

**问题**：未读但需关注：
- 必须用 bcrypt cost ≥ 12 或 Argon2id。
- API Key 不能明文存（推荐 SHA256(salt + key) 或一次性 random + 仅返回一次）。

**建议**：审计 `internal/domain/identity/auth_service.go` 的哈希算法与 SIP 密码 `agents.sip_password_enc`（VARBINARY → 应当是 AES-GCM 而非明文）。

---

## P4 可观测性与运维缺口

### P4-1 没有分布式追踪（OpenTelemetry / Jaeger）

**定位**：`grep -rn "otel\|opentelemetry\|tracing" internal/` 0 结果。

**问题**：一通通话经过 Inbound → IVR → Queue → Ringing → Bridge → Recording → CSAT → CRM → Webhook → AI summary 共 8+ 跳，出问题排查只能拼日志。

**建议**：
- 引入 OTEL SDK，HTTP/SQL/Redis/ESL/AI provider 全装。
- 关键操作创建 span：`call.lifecycle.end`, `acd.dispatch`, `esl.originate`, `llm.summarize`。
- 后端用 Tempo / Jaeger 接 Grafana。

---

### P4-2 没有结构化业务事件日志

**定位**：现有日志为 zerolog，但只在错误路径打。

**建议**：所有领域事件（call_created/answered/ended、agent_state_changed、campaign_dialed）走 `logger.Info().Str("event", ...).Int64(...)` 结构化 → 直接喂 Loki/ELK。

---

### P4-3 没有「饱和度」与「错误预算」SLO 仪表盘

**问题**：Grafana 现有 `ccc-overview.json` 看技术指标，没有业务 SLO（坐席接通率、平均接听时长、IVR 完成率、ACW 超时率）。

**建议**：
- 引入 4 个 SLO 仪表盘：可用性、延迟、饱和度、业务可服务度。
- 配 Alert：呼入失败率 > 1%、IVR 流程超时 > 5%、ACW 超期率 > 10%。

---

### P4-4 没有「灰度发布 / 蓝绿」运维通路

**建议**：FreeSWITCH 是有状态长连接，必须支持 graceful drain（停止接新呼、等存量呼挂断、再下线）。当前 `main.go:594-604` shutdown 只关 HTTP，没等通话与 WebSocket。

---

### P4-5 没有租户级别的「容量水位」面板

**建议**：每租户面板：当前并发呼、未读语音邮箱、IM 在线访客、待处理工单、当前活跃坐席。运营/客户成功团队的核心抓手。

---

## 行业对标建议

> 对标：Genesys Cloud、Avaya OneCloud、Five9、AWS Connect、阿里云联络中心、华为云 CEC

### A. 路由 / ACD 能力

| 能力 | Genesys / AWS Connect 标配 | CCC 当前状态 | 优先级 |
| --- | --- | --- | --- |
| Skills-based routing (多技能、权重) | ✅ | ⚠️ schema 有 routing_policy 但无引擎 | P0 |
| Predictive routing (历史满意度模型) | ✅ (Genesys Predictive Engagement) | ❌ | P1 |
| Best-skill / Best-match algorithm | ✅ | ❌ | P1 |
| Estimated Wait Time (EWT) 公告 | ✅ | ⚠️ IVR node 有 EWTAnnounceInterval 字段但无算法 | P1 |
| Callback in queue (放弃排队回拨) | ✅ | ⚠️ callback_request_repo 存在但无业务接入 | P1 |
| 跨技能组溢出（overflow chain） | ✅ | ❌ | P1 |
| 紧急通话优先级抢占 | ✅ | ❌ | P2 |
| Geo / Time-based routing | ✅ | ⚠️ business_hours 表存在但 IVR 未集成 | P2 |

**建议**：先把 P0-4 的自研 ACD 引擎做出来，把表里的 routing_policy 转为可执行代码；再逐步加 Predictive routing（输入历史 talk_time、CSAT、disposition 训练 LR 模型）。

### B. 自助服务 / 智能化

| 能力 | 业界做法 | CCC 当前 | 优先级 |
| --- | --- | --- | --- |
| Voicebot (LLM 驱动对话) | AWS Lex + Connect | ⚠️ digital_employee 表存在 + LLM gateway 就绪 | P1 |
| 实时坐席辅助 (Agent Assist) | Google CCAI、Cresta、Aircall | ⚠️ AiAssistPanel.tsx 存在但后端 hub 未推 transcript | P1 |
| 知识库 RAG | ✅ | ⚠️ knowledge 模块有但未做向量检索 | P2 |
| 全双工 AI 对话 | 较少 (Aircall AI、阿里通义) | ✅ Phase 11 已有 FullDuplex svc | OK |
| 后处理摘要 (AI Wrap-up) | ✅ | ✅ ai_summary 字段 + AI hooks | OK |
| 实时情绪分析 | ✅ | ⚠️ aianalysis 服务存在但实时性未验证 | P2 |

**建议**：把 `RealtimeTranscriptPanel.tsx` 与 `transcripthub` 真正打通；ASR 推流 → Hub 广播 → AI assist hub 实时摘要 / 推荐话术 / 风险预警。

### C. 渠道融合 (Omnichannel)

| 能力 | 业界做法 | CCC 当前 | 优先级 |
| --- | --- | --- | --- |
| 单一 inbox（多渠道统一队列） | ✅ Five9/Genesys | ⚠️ IM + Voice + Email 各有 hub 但没有统一 conversation 实体 | P1 |
| 渠道转移 (Voice ↔ Chat ↔ Email) | ✅ | ❌ | P1 |
| 上下文继承 (跨渠道 context preservation) | ✅ | ⚠️ customers 表能记录 interactions，但前端无统一时间线 | P1 |
| Social channel (WhatsApp/微信公众号/微博) | ✅ | ⚠️ social_channels 表存在，接入待补 | P2 |

**建议**：抽象 `Conversation` 聚合，跨 Voice/IM/Email/Social 共享 `customer_id + thread_id`，前端 phone workbench 改为 conversation-centric。

### D. WFM (Workforce Management)

| 能力 | 业界做法 | CCC 当前 | 优先级 |
| --- | --- | --- | --- |
| 班次/排班 (Scheduling) | Genesys WFM、NICE IEX | ❌ | P2 |
| 实时坚守度 (Adherence) | ✅ | ⚠️ agent_presence_log 有 sub_state 但缺对比排班 | P2 |
| 预测呼入量 (Forecasting) | ✅ Erlang-C/AI | ❌ | P3 |
| 坐席自助换班 / 申请休假 | ✅ | ❌ | P3 |

**建议**：作为独立子产品（Phase 12+）规划，可在仪表盘里先做 Adherence 简版（在岗时段 / 实际在岗时段对比）。

### E. 合规 / 数据治理

| 能力 | 业界做法 | CCC 当前 | 优先级 |
| --- | --- | --- | --- |
| 录音脱敏 (PCI-DSS Pause/Resume) | ✅ | ❌ | P0（金融业务必备） |
| DNC 自动同步 (FCC/TRAI) | ✅ | ⚠️ dnc_list 表 + cli_policy 在但 dialer 入口未强校验 | P0 |
| 录音水印 (法律证据链) | ✅ | ❌ | P2 |
| 录音多重加密 + KMS | ✅ | ❌ | P1 |
| 跨境数据合规 (数据驻留) | ✅ | ❌ (region 字段缺失) | P2 |

### F. 平台 / SaaS 化能力

| 能力 | 业界做法 | CCC 当前 | 优先级 |
| --- | --- | --- | --- |
| 多 region / 多可用区 | ✅ | ❌ | P2 |
| 自助申请号码 (Number provisioning API) | ✅ | ⚠️ phone_numbers 表存在但无 Twilio/阿里通信适配器 | P2 |
| Open API / Webhook 市场 | ✅ | ⚠️ webhook_config 存在但事件类型有限 | P1 |
| Marketplace / App store | ✅ | ❌ | P3 |
| 计费 / Quota | ✅ | ❌ | P2 |

### G. 开发者体验 / Extensibility

| 能力 | 业界做法 | CCC 当前 | 优先级 |
| --- | --- | --- | --- |
| Flow Designer 公开 SDK | ✅ | ⚠️ IVR 可视化编辑器有但无 npm 包 | P3 |
| Agent Desktop 嵌入 (SDK iframe) | ✅ | ❌ | P2 |
| Browser Plugin (CTI for CRM) | ✅ | ❌ | P2 |
| Public CLI / Terraform Provider | ✅ AWS Connect / Twilio | ❌ | P3 |

---

## 优先级路线图

### 立即修复（一个迭代内，2 周内）

1. **P0-1**：calls 表 schema 与 ORM 字段对齐（单次 ALTER + 实体调整）
2. **P0-2**：拆 `/internal/v1` 路由组 + 服务鉴权中间件（HMAC）
3. **P0-3**：ESL 事件订阅 goroutine（CHANNEL_ANSWER/HANGUP/BRIDGE 最小集）
4. **P0-5**：IMSessionHandler.SendMessage 注入 Hub 广播
5. **P0-6**：main.go 实例化 MinIOClient，接通 Recording Stream/Download
6. **P3-3**：JWT 默认密钥启动检查 + fatal
7. **P3-1 / P3-2**：WebSocket Origin 白名单 + JWT 鉴权

### 短期（1 个月内）

1. **P0-4**：自研 ACD 队列引擎（Redis ZSet + Lua），实现 longest_idle / round_robin 两种策略先跑通
2. **P1-1**：领域事件总线（NATS JetStream），lifecycle 的 post-call hook 解耦
3. **P1-3**：NLS Token 续期 goroutine
4. **P1-5**：max_concurrent_calls 准入控制
5. **P1-11**：ACW 超时自动转 idle
6. **P2-1 / P2-2 / P2-10**：数据库连接池、Call 列表 keyset 分页、业务 metric 接入
7. **P3-5**：录音 / 转写脱敏管线

### 中期（一个季度内）

1. **P0-4 进阶**：5 种 routing_policy 全实现 + EWT 算法 + overflow chain
2. **P1-6**：录音合规告知 IVR 节点
3. **P1-7**：customer_agent_relations 表 + 熟客优先
4. **P1-8**：Predictive dialer 反馈控制
5. **P1-10**：Webhook outbox + DLQ
6. **P2-7**：报表流式导出 + 异步任务
7. **P3-8**：密码 / SIP 密钥安全审计
8. **P4-1 / P4-2 / P4-3**：OTEL 接入 + SLO 仪表盘

### 长期（半年以上）

1. WFM 基础（adherence + scheduling）
2. Agent Assist 实时辅助管线打通
3. Conversation 聚合 + 渠道融合
4. 多 region / 数据驻留
5. Open API + 计费

---

## 附录 A：关键文件与定位汇总

| 主题 | 文件 | 行号 | 简述 |
| --- | --- | --- | --- |
| Schema vs ORM 不匹配 | `migrations/000001_init_schema.up.sql` | 387-435 | calls 表定义 |
| Schema vs ORM 不匹配 | `internal/infrastructure/mysql/call_repo.go` | 19-55 | INSERT/UPDATE 使用错误列名 |
| Inbound JWT 阻塞 | `internal/interfaces/http/router.go` | 177-181, 389 | /api/v1 路由全挂 Auth |
| ESL 事件订阅缺失 | `internal/infrastructure/esl/client.go` | 全文 | 仅 outbound 命令，无 event listener |
| ACD 引擎缺失 | `internal/application/ivr/nodes_routing.go` | 42 | 把队列扔给 mod_callcenter |
| ACD 引擎缺失 | `deploy/freeswitch/conf/autoload_configs/` | 目录 | 无 callcenter.conf.xml |
| IM 广播断链 | `internal/interfaces/http/handler/im_session.go` | 132-150 | SendMessage 不调 Hub |
| MinIO 未实例化 | `cmd/server/main.go` | 全文 | 无 storage.NewMinIOClient 调用 |
| MinIO 未实例化 | `internal/interfaces/http/handler/recording.go` | 49-57 | Stream/Download 返回 501 |
| 连接池配置不足 | `internal/infrastructure/mysql/db.go` | 14-15 | MaxOpen=50, NoLifetime |
| WS 鉴权缺失 | `internal/application/agenthub/hub.go` | 96-100 | URL Query agent_id |
| WS Origin 不校验 | `internal/application/agenthub/hub.go` | 18 | CheckOrigin: true |
| 审计同步阻塞 | `internal/interfaces/http/middleware/audit.go` | 26-37 | 同步 repo.Create |
| Rate Limit 不区分租户 | `internal/interfaces/http/middleware/ratelimit.go` | 10 | 固定 100 |
| NLS Token 不续期 | `cmd/server/main.go` | 258-267 | 启动一次性 Fetch |
| 业务指标 0 调用 | `internal/interfaces/http/middleware/metrics.go` | 36-82 | 定义齐全无 Set/Inc |
| NATS 未使用 | `internal/infrastructure/nats/client.go` | 全文 | 无消费者 |

---

## 附录 B：本轮分析方法论

1. **代码 + Schema 对照**：将 calls 表迁移脚本与 Go 实体字段逐列对比，发现 P0-1 schema 字段错位。
2. **路由表逐项审查**：router.go 全量阅读，识别 P0-2 inbound 鉴权问题。
3. **跨层依赖追踪**：从 main.go 入口反向追踪，找出 NATS/MinIO 仅声明未注入。
4. **关键字 grep**：`OnEvent / CHANNEL_HANGUP / routing_policy / familiar_agent` 等业界关键能力字 grep 验证有无对应实现。
5. **前后端协议对比**：endpoint.ts vs router.go vs phone components，识别 IM 双工链路断裂。
6. **业务上下文回放**：把每个缺陷映射到「真实坐席 / 真实客户」会遇到的场景，明确业务影响。

---

**报告状态**：完。本文不修改任何代码、不生成 PR、不触发 CI。

**建议下一步动作**：
1. 与团队 review 本报告并对 P0/P1 重新打优先级
2. 选择 P0-1 / P0-3 / P0-5 / P0-6 四项作为首个迭代目标（影响最大、改动可控）
3. 把附录 A 的「关键文件定位汇总」作为后续 PR 的 checklist
