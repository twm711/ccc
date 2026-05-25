# CCC 系统行业开发建议（深度版）

> 基于对系统 15 个 application 服务、14 个 domain 模块、12 个 infrastructure 适配器、60+ handler 的逐层分析，对标 Genesys Cloud / Amazon Connect / Five9 等头部产品，提出以下分层建议。

---

## 一、架构与基础设施演进

### 1.1 全渠道统一路由引擎（Omnichannel Routing Engine）

**现状差距**：当前 voice ACD（`application/acd`）和 IM Router（`application/imrouter`）是两套独立的路由实现。Voice 基于 Redis ZSET 排队 + ticker 轮询，IM 基于 skill group 成员遍历。两者的路由策略（longest-idle、round-robin、familiar）实现分散，无法跨渠道统一调度。

**行业标准**：Genesys PureCloud 和 Amazon Connect 采用统一的 Routing Engine，一个坐席可以同时处理 1 路语音 + 3 路 IM，系统根据"媒体容量"（media capacity）统一排队分配。

**建议实施路径**：
```
Phase 1: 统一 Agent Capacity 模型
  - AgentPresence 增加 MediaCapacity{voice: 1, im: 3, email: 5}
  - 路由决策时检查各渠道剩余容量，而非仅检查 idle 状态

Phase 2: 统一排队层
  - 将 IM 等待队列也放入 Redis ZSET（复用 acd 的 queue 机制）
  - 添加 media_type 维度到 queueKey: acd:queue:{sg}:{media}

Phase 3: 跨渠道优先级
  - 支持 voice > im > email 的全局优先级配置
  - 当语音队列积压时，暂缓 IM 分配以释放坐席处理语音
```

### 1.2 微服务拆分准备（Microservice Readiness）

**现状**：单体部署（`cmd/server/main.go` 约 900 行，所有服务在同一进程），通过 NATS JetStream 和 Kafka 已具备异步通信基础。

**建议**：不急于拆分，但做好以下准备：
- **事件驱动完善**：当前只有 `ccc.call.ended` / `ccc.call.answered` 两个事件走 NATS。建议将所有状态变更（agent.status_changed、im.session.created、campaign.started）统一发布到 NATS，使未来拆分时各服务只需消费事件流
- **接口隔离**：`lifecycle.Service` 当前直接依赖 10+ 个具体 service 指针。用 interface 隔离后，拆分时只需替换 implementation（本轮已开始此工作：SMSSender、CallerLookup 等）
- **数据库隔离**：当前所有表在同一 MySQL schema。建议按 bounded context 分 schema（call_*, identity_*, crm_*, campaign_*），为未来分库做准备

### 1.3 高可用与水平扩展

**现状差距**：
- ACD dispatcher 是单 goroutine（注释明确提到"single goroutine"），多实例部署时需手动分配 skill group 子集
- Dashboard Refresher 每 10 秒遍历所有租户，随租户数增长线性变慢
- Callback Scheduler 每 30 秒全表扫描 pending callbacks

**建议**：
- **ACD 分布式锁**：用 Redis SETNX 对每个 skill group 加分布式锁，多实例自动瓜分负载（类似 agentClaimPrefix 的模式扩展到 sgDispatchLock）
- **Dashboard 分片刷新**：按 `tenant_id % instance_count` 分片，每个实例只刷新自己负责的租户
- **Callback 延迟队列**：用 Redis ZSET（score = scheduled_at）替代全表扫描，按时间窗口精确拉取

---

## 二、AI 与智能化深化

### 2.1 实时质检（Real-time QA）

**现状**：QA（`application/aianalysis`）仅支持 post-call 质检——通话结束后对转写文本分析。ASR 转写（`infrastructure/llm/aliyun_asr_provider`）虽然支持流式，但质检逻辑未接入实时流。

**建议**：
- **流式质检管道**：在 `transcripthub` 的 WebSocket 转写流中插入实时分析节点
  - 敏感词命中（辱骂、承诺违规）→ 立即通知主管
  - 情绪波动检测 → 自动弹出话术推荐
  - 合规检查（如保险行业必须念免责条款）→ 实时提醒
- **实时评分面板**：在 SupervisorPanel 增加通话中实时评分卡
- **阈值联动**：敏感词命中 N 次 → 自动触发 `CallType=BARGE`（主管强插）

### 2.2 智能 IVR 升级

**现状**：IVR 引擎（`application/ivr`）支持 20 种节点类型，包括 ASR、DigitalEmployee、HTTP Request 等。但缺少以下行业关键能力：

**建议新增节点类型**：
- `NodeNLU`：自然语言理解节点，支持意图识别（而非仅 DTMF / ASR 转文字）
- `NodeSentimentGate`：根据情绪分值分流（高愤怒 → VIP 通道，正常 → 常规排队）
- `NodeCustomerLookup`：在 IVR 阶段查 CRM，根据客户等级（VIP/SVIP）自动提升 ACD 优先级
- `NodeQueuePosition`：播报排队位置和预计等待时间（用 `acd:queue:{sg}` 的 ZSET 长度估算）

### 2.3 Agent Copilot（坐席 AI 助手）增强

**现状**：已有 `AiAssistPanel`（前端）、`imassist`（后端）、`ScriptRecommendPanel` 提供基础辅助。

**建议深化方向**：
- **下一步最佳动作（Next Best Action）**：基于对话上下文 + CRM 客户画像，实时推荐"建议转 VIP 专线"/"推荐产品 X"/"升级工单"
- **自动填单**：通话中 AI 实时提取关键信息（姓名、单号、诉求）自动填充 SessionInfoTemplate 和工单字段
- **知识库 RAG**：当前 KnowledgeArticle 仅支持关键词搜索。接入向量检索（如 Milvus / pgvector）后，坐席提问时可语义匹配最相关文章段落

### 2.4 数字员工（Digital Employee）增强

**现状**：已有 `CommAgent`（自主对话代理）、VoiceCloning、FullDuplex 等 AI 能力。

**建议**：
- **多轮对话状态机**：当前 `HandleConversationTurn` 是无状态的每轮独立调用。建议引入对话状态机（slot-filling model），追踪已收集/待收集的信息槽位
- **无缝人机接力**：转人工时自动将 AI 收集的结构化信息（客户姓名、问题类型、已尝试方案）注入坐席 ScreenPop，减少客户重复
- **语音情感自适应**：根据客户语气自动调整 TTS 语速和语调（CosyVoice 支持 emotion 参数）

---

## 三、运营精细化

### 3.1 WFM（Workforce Management 排班管理）

**现状缺失**：无排班模块。坐席上线/离线完全手动。

**行业必备功能**：
```
1. 历史数据预测
   - 基于 CallTrend（已有）按半小时粒度预测未来 N 天呼叫量
   - Erlang C 公式计算所需坐席数

2. 排班生成
   - 输入：预测呼叫量 + 坐席可用性 + 法规约束（连续工作 ≤ 4h、间隔 ≥ 30min）
   - 输出：最优排班表（最小化人力成本同时满足 SLA 目标）

3. 实时监控 & 调度
   - 将 DashboardOverview 的 ServiceLevel20s 与排班预估对比
   - SL 低于阈值 → 自动通知 off-duty 坐席紧急上线（钉钉/企微推送）

4. 考勤合规
   - 自动记录实际登录时间 vs 排班时间
   - 迟到/早退/加班统计，对接 HR 系统
```

### 3.2 SLA 分层报警体系

**现状**：ServiceLevel20s 已计算但无报警机制。

**建议**：
```yaml
alarm_rules:
  - metric: service_level_20s
    thresholds:
      - level: warning
        condition: "< 80%"
        action: notify_manager  # 钉钉/企微推送
      - level: critical
        condition: "< 60%"
        action: [notify_manager, auto_recall_agents]  # 召回休息坐席
      - level: emergency
        condition: "< 40%"
        action: [notify_director, enable_overflow_routing]  # 跨技能组溢出

  - metric: avg_wait_sec
    thresholds:
      - level: warning
        condition: "> 30s"
      - level: critical
        condition: "> 60s"

  - metric: longest_wait_sec
    thresholds:
      - level: critical
        condition: "> 120s"
        action: priority_boost  # 等待超久的来电自动提升排队优先级
```

### 3.3 自动工单联动

**现状**：工单（`domain/ticket`）和通话/IM 会话之间的关联是手动的（Ticket.CallID 字段可选）。

**建议自动化场景**：
- CSAT 低分（≤ 2 分）→ 自动创建投诉工单，关联 call_id + recording URL
- IM 会话超时未解决（超过 SLA）→ 自动升级为工单
- 客户回拨 3 次同一号码 → 自动创建升级工单（识别为重复来电未解决）
- 通话中 AI 检测到客户要求投诉 → 自动创建工单并通知主管

### 3.4 外呼合规增强

**现状**：已有 DNC 名单、呼叫时段限制（09:00-20:00 MIIT 合规）、CLI Policy。

**建议补充**：
- **频率限制**：同一号码每天最多被呼 N 次（跨 campaign 累计）。在 `dialCase` 前增加 Redis 频率计数
- **接通率监控**：Ofcom 规定 predictive dialer 弃呼率不得超过 3%。当前 `MaxAbandonRate` 支持此功能但未提供运营看板——建议在 CampaignLiveDashboard 实时展示
- **录音告知**：部分地区法规要求在通话接通前播放"本次通话将被录音"。已有 `RecordingAnnounce` 配置但需确认 IVR 流程中是否强制集成
- **退订机制**：外呼接通后提供 DTMF 选项"按 9 加入勿扰名单"，自动写入 DNC

---

## 四、数据与报表增强

### 4.1 BI 数据仓库

**现状**：报表数据实时计算（MySQL 聚合 + Redis 缓存），适合实时大屏但不适合历史趋势分析。

**建议**：
- **CDR 归档**：每日凌晨将 completed calls 归档到 ClickHouse / StarRocks，保留原始明细
- **维度模型**：构建星型 schema（fact_call, dim_agent, dim_skill_group, dim_tenant, dim_time）
- **预聚合**：Agent 日报/周报/月报预计算存储，避免每次查询全表扫描
- **Kafka → BI**：当前 Kafka Producer 只发 CDR。建议扩展为所有分析事件的 sink，接入 Flink/Spark 做流式 ETL

### 4.2 客户旅程分析（Customer Journey）

**现状**：CustomerInteraction 记录了 call/ticket/im 三种渠道交互，但无跨渠道关联分析。

**建议**：
- 构建 `CustomerJourney` 视图：按时间线聚合同一客户的所有交互（来电 → IM 咨询 → 工单 → 回访电话）
- 识别关键转折点：哪一步导致客户流失？哪一步提升了满意度？
- 前端展示：在 CustomerPage 增加"客户旅程"时间轴组件

### 4.3 坐席绩效评分

**现状**：AgentReport 已有丰富指标（通话量、平均通话时长、ACW 时长、SL20s、利用率等）。

**建议新增**：
- **综合绩效分** = f(接通率, 满意度, 首次解决率, SLA, 利用率) 加权计算
- **同组排名**：同 SkillGroup 内坐席 KPI 横向对比
- **趋势分析**：周环比/月环比，自动标记绩效下滑的坐席
- **对接激励系统**：绩效分 → 积分/排行榜 → gamification

---

## 五、安全与合规

### 5.1 数据安全增强

**现状**：已有录音加密（`infrastructure/crypto/recording.go`）、API 鉴权（JWT + RBAC）、审计日志。

**建议补充**：
- **PII 脱敏**：转写文本和 CRM 数据中的手机号、身份证号自动脱敏后存储。当前 `pkg/redact` 已有基础实现，需在更多出口点（API 响应、导出、日志）统一拦截
- **数据保留策略**：按租户配置录音/CDR/转写的保留天数（已有 `RecordingRetentionDays`），到期自动清理 MinIO + MySQL
- **字段级权限**：不同角色看到不同字段（如 agent 看不到客户完整手机号，manager 可以）
- **GDPR/个人信息保护法**：支持"被遗忘权"——客户要求删除个人数据时，一键清除 CRM + 录音 + 转写

### 5.2 多租户隔离加固

**现状**：基于 `tenant_id` 的行级隔离（每条 SQL WHERE tenant_id = ?）。

**建议**：
- **租户级 Redis key namespace**：当前 dashboard key 为 `dashboard:{tenant_id}`，但 ACD 的 `acd:active_sg` 是全局 SET——确保不会跨租户泄漏技能组
- **API 层自动注入 tenant_id**：在中间件统一从 JWT 提取 tenant_id 注入 context，service 层不再信任前端传入的 tenant_id
- **资源配额**：按租户限制最大坐席数（已有 MaxAgents）、最大并发通话数（已有 MaxConcurrentCalls）、最大 API QPS（已有 APIRateLimitPerSec）。建议增加存储配额（录音空间）和 AI 调用配额

---

## 六、前端与用户体验

### 6.1 坐席工作台统一

**现状**：AgentPhoneBar（通话控制）+ ScreenPopPanel + AiAssistPanel + ScriptRecommendPanel + RealtimeTranscriptPanel 分散在多个组件中。

**建议**：
- **统一工作台布局**：左侧客户信息 + 中间对话/通话 + 右侧 AI 辅助，一屏展示所有关键信息
- **多会话切换**：坐席同时处理 1 路语音 + N 路 IM 时，支持 tab 切换不丢失上下文
- **快捷键体系**：F5 接听、F6 挂断、F7 保持、F8 转接——减少鼠标操作

### 6.2 主管实时监控增强

**现状**：SupervisorPanel 支持监听/耳语/强插。DashboardPage 展示实时数据。

**建议**：
- **墙板模式（Wallboard）**：大屏全屏展示关键 KPI（今日通话量、排队数、SL、弃呼率），适合呼叫中心大厅投屏
- **预警热力图**：按技能组展示排队压力热力图，红色 = 排队严重
- **一键调度**：主管在热力图上直接拖拽坐席到其他技能组临时增援

### 6.3 WebRTC 通话质量

**现状**：已有 `webrtc_quality` handler 收集 WebRTC 质量数据。

**建议**：
- **MOS 分实时展示**：在通话中展示网络质量指标（RTT、jitter、packet loss → MOS 估算）
- **自动降级策略**：网络差时自动降低音频码率或切换到 PSTN 回呼
- **质量趋势报告**：按坐席/地区汇总 WebRTC 质量，识别网络问题热点

---

## 七、集成与开放平台

### 7.1 Webhook 增强

**现状**：Webhook 支持事件推送 + HMAC 签名 + 重试（max 3 次）。

**建议**：
- **事件类型扩展**：当前仅 call.* 事件。建议覆盖全部 domain 事件（agent.login、campaign.completed、ticket.created、csat.submitted）
- **订阅过滤**：允许 webhook 配置只接收特定事件类型（而非全部）
- **投递可观测性**：webhook_delivery_log 已有，建议增加前端 WebhookPage 的投递成功率统计图表
- **Dead Letter Queue**：3 次重试全部失败后进入 DLQ，管理员可手动重投

### 7.2 Open API 标准化

**建议**：
- **OpenAPI 3.0 规范**：为所有 handler 生成 Swagger 文档，方便第三方集成
- **API 版本管理**：URL 路径加 `/v1/`，为未来破坏性变更预留空间
- **SDK 生成**：基于 OpenAPI spec 自动生成 Python/Java/Node SDK
- **沙箱环境**：提供测试租户 + mock 电话通道，让集成商安全调试

### 7.3 第三方 CRM 对接

**现状**：CRM 内置（Customer/Interaction/CustomField），但无法对接 Salesforce/HubSpot 等外部 CRM。

**建议**：
- **CRM Connector 框架**：定义 `CRMConnector` 接口（LookupCustomer、CreateLead、SyncInteraction），实现 Salesforce/HubSpot/纷享销客 适配器
- **Screen Pop 数据源扩展**：当前 ScreenPop 仅查内置 CRM。支持同时查外部 CRM 并合并展示
- **双向同步**：CCC 创建的客户/工单 → 同步到外部 CRM；外部 CRM 更新 → webhook 回写 CCC

---

## 八、实施优先级建议

| 优先级 | 建议项 | 预期收益 | 工作量 |
|--------|--------|----------|--------|
| P0 | SLA 分层报警 | 防止 SL 崩溃无人知晓 | 2-3 天 |
| P0 | 外呼频率限制 | 合规风险规避 | 1-2 天 |
| P0 | PII 脱敏统一 | 数据安全合规 | 3-5 天 |
| P1 | 全渠道统一路由 Phase 1 | 坐席利用率提升 20-30% | 1-2 周 |
| P1 | 实时质检 MVP | 降低投诉风险 | 1-2 周 |
| P1 | 自动工单联动 | 减少人工操作 | 3-5 天 |
| P1 | WFM 排班预测 | 优化人力成本 | 2-3 周 |
| P2 | 知识库 RAG | 坐席效率提升 | 1-2 周 |
| P2 | 客户旅程分析 | 运营洞察 | 1-2 周 |
| P2 | BI 数据仓库 | 历史分析能力 | 2-3 周 |
| P2 | Open API + SDK | 生态扩展 | 2-3 周 |
| P3 | 微服务拆分 | 团队独立交付 | 持续演进 |
| P3 | 统一工作台重构 | 坐席体验 | 2-3 周 |

---

## 九、技术债务清理建议

1. **测试覆盖率**：当前仅 domain 层有 unit test。建议为 application 层（特别是 lifecycle、acd、dialer）补充集成测试
2. **CI 修复**：main 分支 CI 持续失败，需排查根因并修复（可能是 Go 版本 1.25 与 GitHub Actions setup-go 兼容问题）
3. **配置外置**：`main.go` 中硬编码的值（如 callback 30s 间隔、ACD 500ms 轮询）应通过 config 注入
4. **错误处理**：部分地方用 `_ = repo.Update(ctx, ...)` 吞掉错误。关键路径应记录并上报
5. **Graceful Shutdown**：确保所有 background goroutine 在 SIGTERM 时优雅退出（当前 hubCtx cancel 已覆盖大部分）
