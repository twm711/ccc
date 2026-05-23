# 第二轮 Review — 拨号模式实现细节 + 高并发性能优化 + 遗漏检查

---

## 一、拨号模式实现细节分析

### 当前状态

v5 定义了 4 种外呼模式（Phase 6），campaigns 表有 `dialing_mode` 枚举和 `ratio_multiplier` 字段。
实施计划 Phase 6 有任务 6.1-6.4 + TDD 用例覆盖 4 种模式。

### 🔴 遗漏项（共 6 项）

#### 1. campaigns 表缺失关键字段

当前 campaigns 表只有 `ratio_multiplier` 控制拨号比率，但各模式需要不同的控制参数：

```sql
-- 建议新增字段
ALTER TABLE campaigns ADD COLUMN preview_timeout_sec  INT UNSIGNED DEFAULT 30 COMMENT 'Preview模式：坐席预览超时秒数';
ALTER TABLE campaigns ADD COLUMN max_abandon_rate     DECIMAL(4,2) DEFAULT 3.00 COMMENT 'Predictive模式：最大放弃率阈值(%)，FCC/工信部合规';
ALTER TABLE campaigns ADD COLUMN concurrent_limit     INT UNSIGNED DEFAULT 0 COMMENT '本活动最大并发呼叫数，0=不限';
ALTER TABLE campaigns ADD COLUMN timezone             VARCHAR(64) DEFAULT 'Asia/Shanghai' COMMENT '调度时区';
ALTER TABLE campaigns ADD COLUMN schedule_days        SET('mon','tue','wed','thu','fri','sat','sun') DEFAULT 'mon,tue,wed,thu,fri' COMMENT '允许外呼的星期';
```

**问题：**
- `schedule_start/end TIME` 只支持单个每日时间窗口，无法表达"工作日 9:00-12:00, 14:00-18:00"这种多时段
- 无 timezone 字段，时间窗口无法跨时区使用
- 无并发限制字段，predictive 模式可能耗尽所有 FreeSWITCH 资源

#### 2. 4 种模式的行为规范未在文档中定义

当前仅有 TDD 测试名和任务描述，缺乏具体行为规范：

| 模式 | 行为定义（建议补充） |
|---|---|
| **Predictive** | 系统根据坐席可用预测自动批量拨号。`ratio_multiplier` 控制每空闲坐席拨出比率（如1.5=每个空闲坐席拨1.5通）。实时监控放弃率，超过 `max_abandon_rate` 时自动降速。需 Erlang-C 或滑动窗口算法预测坐席可用时间。|
| **Preview** | 系统推送案例到坐席屏幕，坐席查看客户资料后手动决定拨号/跳过。超过 `preview_timeout_sec` 未操作则自动跳过并推送下一个。需前端 UI：案例卡片 + "拨号/跳过" 按钮。|
| **Progressive** | 坐席进入空闲状态时，系统自动拨出 1 通（1:1 固定比率）。坐席无需手动操作。接通后自动桥接。|
| **Power** | 类似 Progressive 但使用固定高比率（如 `ratio_multiplier`=2.0-3.0），每个空闲坐席同时拨多通。第一个接通的桥接给坐席，其余挂断。高放弃率风险。|

#### 3. Predictive 模式缺乏自适应算法规范

预测式外呼是最复杂的模式，需要：

- **输入信号：** 当前空闲坐席数、平均通话时长、平均振铃时长、接通率
- **预测逻辑：** 预计 N 秒后将有 M 个坐席空闲 → 拨出 M × ratio 通电话
- **反馈回路：** 实时监控放弃率，动态调整 ratio（放弃率 > 阈值 → 降速，< 阈值 → 加速）
- **合规：** 中国工信部对机器外呼有规定，需支持放弃率阈值配置

**建议：** 在实施计划 Phase 6 任务 6.3 中补充为：
```
6.3a Application: Predictive Dialer — 坐席可用预测引擎 (滑动窗口/Erlang-C)
6.3b Application: Predictive Dialer — 实时放弃率监控 + ratio 自适应调节
6.3c Application: Predictive Dialer — 合规限速 (max_abandon_rate 触发降速)
```

#### 4. Preview 模式缺乏前端交互规范

Preview 模式需要专属 UI 交互，但 Phase 6 前端任务 6.10 只写了"批量外呼（活动CRUD/案例导入/运行监控/实时统计）"，没有：

- 坐席工作台增加 "案例预览卡片" 组件
- "拨号 / 跳过 / 备注" 操作按钮
- 预览倒计时显示

**建议：** Phase 6 前端任务拆分：
```
6.10a 前端: 批量外呼管理页（活动CRUD/案例导入/运行监控/实时统计）
6.10b 前端: 坐席工作台 — Preview 模式案例预览卡片 + 拨号/跳过操作
6.10c 前端: Campaign 实时大屏（并发数/接通率/放弃率/进度条）
```

#### 5. campaign_cases 表缺乏结果追踪字段

```sql
-- 建议新增
ALTER TABLE campaign_cases ADD COLUMN agent_user_id BIGINT UNSIGNED COMMENT '处理坐席';
ALTER TABLE campaign_cases ADD COLUMN duration_sec   INT UNSIGNED COMMENT '通话时长';
ALTER TABLE campaign_cases ADD COLUMN disposition_code VARCHAR(64) COMMENT '结案代码';
ALTER TABLE campaign_cases ADD COLUMN next_attempt_at TIMESTAMP NULL COMMENT '下次重拨时间(重试间隔计算)';
```

#### 6. 缺乏 Campaign API WebSocket 实时推送

Campaign 运行监控需要实时数据（每秒更新的并发数/接通率/放弃率），但 Phase 6 API 只有 REST：
```
GET /api/v1/campaigns/{id}/stats  # 轮询式
```

**建议补充：**
```
WS  /api/v1/ws/campaigns/{id}/live    # WebSocket 实时推送
```

---

## 二、高并发性能优化分析

### 目标：100+ 租户, 500+ 坐席, 1000+ 并发通话

### 🔴 架构层面缺失（5 项）

#### 1. Go 应用层并发规范缺失

当前规划中没有定义：

| 组件 | 需要规范 | 建议值 |
|---|---|---|
| MySQL 连接池 | `max_open_conns` / `max_idle_conns` / `conn_max_lifetime` | 50/25/5min |
| Redis 连接池 | `pool_size` / `min_idle_conns` | 100/20 |
| ESL 连接管理 | 单连接 vs 连接池 | 3-5 连接池 + 命令队列 |
| HTTP Server | `MaxConcurrentStreams` / `ReadTimeout` | 1000/30s |
| Worker Pool | IVR 节点执行/录音上传/Webhook 投递 | 可配置大小 |

**建议：** Phase 0 的 `cmd/server/main.go` 初始化中定义连接池参数，从配置文件读取。

#### 2. ESL 连接高可用策略缺失

v5 Part 7.3 说 "ESL 连接数 ~3（每个 Go 实例一个持久连接）"，但：

- **单连接瓶颈：** 1000 并发通话的控制命令全走一个 ESL TCP 连接
- **连接断开：** 如果 ESL 连接中断，所有通话控制失效
- **多 FreeSWITCH：** Phase 10 要多 FS 实例，但控制平面如何路由到正确的 FS？

**建议架构：**
```
Go App → ESL Connection Pool (per FreeSWITCH instance)
       → ESL Router (call_id → FS instance mapping, Redis 存储)
       → Circuit Breaker (3次失败 → 熔断 30s → half-open 探测)
       → Auto-reconnect (指数退避, max 30s)
```

**建议：** 在 Phase 1 任务 1.5 "FreeSWITCH ESL 连接器" 中补充：连接池 + 重连 + 熔断 + 多实例路由。

#### 3. 写入热点缓解策略缺失

1000 并发通话每秒产生的 MySQL 写入：

| 表 | 写入频率 | 量级 |
|---|---|---|
| call_events | 每次状态变更 | ~2000/s (每通话平均2次/s) |
| ivr_tracking | IVR 节点遍历 | ~500/s |
| agent_presence_log | 坐席状态变更 | ~100/s |
| queue_snapshots | 定期采样 | ~50/s |

**总计约 2500-3000 次/s 写入，MySQL 单实例可承受但余量不足。**

**建议：**
- `call_events` 和 `ivr_tracking` 先写入 Kafka/NATS → 异步批量写入 MySQL（每 500ms 或每 100 条 batch insert）
- `queue_snapshots` 保留策略: 7 天，定时清理
- Phase 10 分区前，在 Phase 4 (报表阶段) 就实施 `calls` 表的 RANGE 分区

#### 4. WebSocket 扩展策略缺失

500+ 坐席每人一个 WebSocket 连接，需要：

- **跨实例广播：** 管理员在 Go 实例 A 看大屏，坐席状态变更发生在实例 B → 需要 Redis PubSub 或 NATS 跨实例同步
- **连接管理：** 心跳检测（30s）、断线重连、连接数限制
- **粘性会话：** 如果使用多 Go 实例，WebSocket 需要 sticky session 或 JWT-based routing

**建议：** Phase 3 (坐席状态) 或 Phase 4 (实时监控) 中明确 WebSocket 跨实例方案。NATS 已在架构中，直接用 NATS 做跨实例事件广播。

#### 5. FreeSWITCH 资源隔离缺失

多租户共享 FreeSWITCH，但无隔离：

- 租户 A 发起 Predictive Campaign (500 并发外呼) → 耗尽所有 FS 资源 → 租户 B 无法接听呼入
- 需要 per-tenant 的 FreeSWITCH session 限额

**建议：**
- tenant_settings 中使用现有 `api_rate_limit_per_sec` 类似的方式，增加：
  ```sql
  ALTER TABLE tenant_settings ADD COLUMN max_concurrent_calls INT UNSIGNED NOT NULL DEFAULT 100;
  ```
- Go 层在 ESL 发起呼叫前检查 Redis 中该租户当前活跃通话数

### 🟡 需确认的性能设计（3 项）

#### 6. ACD 路由引擎性能

当前 Redis SORTED SET 用于排队，但 5 种路由策略（轮询/最近最少/最长空闲/技能加权/熟人模式）的实现细节未定义：

| 策略 | Redis 实现 | 性能 |
|---|---|---|
| 轮询 (round_robin) | LIST + RPOPLPUSH | O(1) |
| 最长空闲 (most_idle) | SORTED SET (score=idle_since) | O(log N) |
| 最近最少 (least_recent) | SORTED SET (score=last_call_end) | O(log N) |
| 技能加权 (skill_weighted) | 多 SORTED SET + 加权选择 | O(N×log M) |
| 熟人模式 (familiar) | 查询 customer_interactions → 匹配坐席 | O(N) + MySQL |

**熟人模式** 需要查 MySQL (最近 `familiar_agent_days` 天内服务过该客户的坐席)，延迟较高。

**建议：** 在 Phase 1 IVR 引擎的 `transfer_to_agent` 节点实现中，明确各策略的 Redis 数据结构和查询方式。熟人模式可用 Redis 缓存最近交互 (key: `familiar:{tenant}:{customer_phone}` → agent_ids)。

#### 7. 录音存储 I/O

1000 并发通话 = 1000 个同时写入的录音文件（WAV ~128KB/s 每通话）:

- **总 I/O: ~125 MB/s 持续写入**
- 本地磁盘 SSD 可承受，HDD 有风险
- FreeSWITCH 直接写本地 → Go 后台上传 MinIO

**建议：**
- 录音先写 FreeSWITCH 本地 → 通话结束后异步上传 MinIO
- 上传 worker pool 大小：50 并发（可配置）
- 磁盘空间告警：80% 使用率告警，90% 停止新录音

#### 8. Prometheus 监控应从 Phase 0 开始

当前 Prometheus + Grafana 在 Phase 10（最后阶段），但性能问题需要从早期就有可观测性。

**建议：**
- Phase 0 加入：`/metrics` endpoint (Prometheus HTTP exporter)
- Phase 0 加入：结构化日志 (zerolog/zap, JSON format)
- Phase 1 加入：ESL 连接状态、通话并发数、IVR 节点执行耗时 metrics
- Phase 4 加入：Grafana dashboard 模板

---

## 三、其他遗漏

### 🟡 数据模型细节（3 项）

#### 9. tenant_settings 缺少 max_concurrent_calls

上面第 5 项已提到。`tenant_settings` 已有 `max_queue_size` 和 `api_rate_limit_per_sec`，但缺少并发通话限额。

#### 10. calls 表缺少 campaign 关联

当前 `calls` 表没有 `campaign_id` 和 `campaign_case_id` 字段，无法关联外呼通话到具体活动和案例：

```sql
ALTER TABLE calls ADD COLUMN campaign_id      BIGINT UNSIGNED COMMENT '所属活动';
ALTER TABLE calls ADD COLUMN campaign_case_id BIGINT UNSIGNED COMMENT '所属案例';
```

#### 11. agents 表缺少 campaign 分配字段

当前无法限制哪些坐席参与哪些 Campaign。可以通过 skill_group 间接关联（campaign 绑定 skill_group），但如果一个坐席在多个技能组，且只有部分技能组参与 campaign，需要更细的控制。

**建议：** 通过 `campaigns.skill_group_id` 关联即可（当前已有），不需新增字段。坐席参与 campaign = 坐席所在技能组被 campaign 引用。这是行业通用做法。✓ 不需改动。

### 🟡 实施计划细节（2 项）

#### 12. Phase 1 退出标准缺乏性能基准

Phase 1 是第一次接入 FreeSWITCH，应该建立基准：
```
- [ ] 单通话端到端延迟 < 500ms (IVR接通到媒体播放)
- [ ] 10 并发通话无异常
```

#### 13. Phase 10 负载测试缺乏具体场景

Task 10.3 "负载测试（1000+并发, k6/Locust）" 太粗略。建议拆分：
```
10.3a 呼入压测：1000 并发呼入 → IVR → ACD → 坐席接听 → 录音
10.3b 外呼压测：Predictive Campaign 500 并发外呼
10.3c 混合压测：500 呼入 + 500 外呼 同时运行
10.3d WebSocket 压测：500 坐席 + 10 管理员大屏 同时在线
10.3e 报表压测：100 万条 CDR 的报表查询 < 5s
```

---

## 四、修复建议汇总

| # | 类型 | 修改位置 | 说明 |
|---|---|---|---|
| 1 | 数据模型 | v5 campaigns 表 | 新增 preview_timeout_sec, max_abandon_rate, concurrent_limit, timezone, schedule_days |
| 2 | 文档 | 实施计划 Phase 6 | 补充 4 种拨号模式行为规范定义 |
| 3 | 任务拆分 | 实施计划 Phase 6 任务 6.3 | 拆分 Predictive Dialer 为 3 个子任务 |
| 4 | 前端 | 实施计划 Phase 6 任务 6.10 | 拆分前端任务，增加 Preview 卡片 + Campaign 实时大屏 |
| 5 | 数据模型 | v5 campaign_cases 表 | 新增 agent_user_id, duration_sec, disposition_code, next_attempt_at |
| 6 | API | 实施计划 Phase 6 | 新增 Campaign WebSocket 实时推送 endpoint |
| 7 | 架构 | 实施计划 Phase 0/1 | 补充连接池/Worker Pool 规范 |
| 8 | 架构 | 实施计划 Phase 1 任务 1.5 | ESL 连接池 + 重连 + 熔断 + 多实例路由 |
| 9 | 性能 | 实施计划 Phase 1/4 | call_events/ivr_tracking 异步批量写入 |
| 10 | 架构 | 实施计划 Phase 3/4 | WebSocket 跨实例方案 (NATS 广播) |
| 11 | 数据模型 | v5 tenant_settings 表 | 新增 max_concurrent_calls |
| 12 | 数据模型 | v5 calls 表 | 新增 campaign_id, campaign_case_id |
| 13 | 可观测 | 实施计划 Phase 0 | 加入 /metrics + 结构化日志 |
| 14 | 退出标准 | 实施计划 Phase 1 | 加入性能基准 |
| 15 | 任务拆分 | 实施计划 Phase 10 任务 10.3 | 拆分为 5 个具体压测场景 |

---

## 结论

拨号模式和高并发两个领域各发现 6 个和 5 个待补充项。核心问题：

1. **Predictive Dialer 是整个系统最复杂的子系统之一**，当前任务粒度过粗，缺乏自适应算法和合规控制
2. **1000 并发下的 MySQL 写入热点** 需要异步缓冲策略
3. **ESL 单连接瓶颈** 在 Phase 1 就需要解决，不能等到 Phase 10
4. **可观测性** 应该从 Phase 0 开始，而不是 Phase 10 最后补

建议：确认后直接修复到实施计划和 v5 文档中。
