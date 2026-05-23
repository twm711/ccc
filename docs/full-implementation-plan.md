# CCC 全阶段实施规划 (Phase 0-10)

> 遵循 DDD 领域驱动设计 + 核心域 TDD + Clean Architecture  
> 每个 Phase 包含：限界上下文/聚合/领域事件/TDD 范围/任务分解/API 路由/前端页面/退出标准

---

## 全局 DDD 限界上下文总图

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              CCC Platform                                        │
│                                                                                   │
│  ┌────────────────┐ ┌────────────────┐ ┌────────────────┐ ┌──────────────────┐  │
│  │ Identity BC    │ │ Call BC        │ │ Routing BC     │ │ Telephony BC     │  │
│  │ (核心域)        │ │ (核心域)        │ │ (核心域)        │ │ (支撑域)          │  │
│  │                │ │                │ │                │ │                  │  │
│  │ Tenant         │ │ Call           │ │ IVRFlow        │ │ Carrier          │  │
│  │ User/Agent     │ │ CallEvent      │ │ IVRFlowVersion │ │ SIPTrunk         │  │
│  │ SkillGroup     │ │ IVRTracking    │ │ RoutingRule    │ │ SIPTrunkGroup    │  │
│  │ AgentPresence  │ │ Recording      │ │ CLIPolicy      │ │ PhoneNumber      │  │
│  │                │ │ QueueSnapshot  │ │                │ │                  │  │
│  │                │ │ CallbackReq    │ │                │ │                  │  │
│  │                │ │ WebRTCQuality  │ │                │ │                  │  │
│  └────────────────┘ └────────────────┘ └────────────────┘ └──────────────────┘  │
│                                                                                   │
│  ┌────────────────┐ ┌────────────────┐ ┌────────────────┐ ┌──────────────────┐  │
│  │ Campaign BC    │ │ CRM BC         │ │ Ticket BC      │ │ IM BC            │  │
│  │ (核心域)        │ │ (核心域)        │ │ (支撑域)        │ │ (支撑域)          │  │
│  │                │ │                │ │                │ │                  │  │
│  │ Campaign       │ │ Customer       │ │ TicketCategory │ │ IMChannel        │  │
│  │ CampaignCase   │ │ CustomerPhone  │ │ TicketTemplate │ │ IMSession        │  │
│  │                │ │ CustInteraction│ │ Ticket         │ │ IMMessage        │  │
│  │                │ │                │ │ TicketComment  │ │                  │  │
│  └────────────────┘ └────────────────┘ └────────────────┘ └──────────────────┘  │
│                                                                                   │
│  ┌────────────────┐ ┌────────────────┐ ┌────────────────┐ ┌──────────────────┐  │
│  │ AI BC          │ │ Configuration  │ │ Operation BC   │ │ Platform BC      │  │
│  │ (支撑域)        │ │ BC (通用域)     │ │ (通用域)        │ │ (支撑域)          │  │
│  │                │ │                │ │                │ │                  │  │
│  │ DigitalEmployee│ │ BreakReason    │ │ AudioFile      │ │ Auth/JWT         │  │
│  │ DEScene        │ │ DispositionCode│ │ BusinessHours  │ │ RateLimit        │  │
│  │ KnowledgeBase  │ │ CustomFieldDef │ │ Voicemail      │ │ AuditLog         │  │
│  │ AgentScript    │ │ CallTag        │ │                │ │                  │  │
│  │ Perf.Scorecard │ │ AutoTagRule    │ │                │ │                  │
│  │ AnnotationTask │ │                │ │                │ │                  │  │
│  └────────────────┘ └────────────────┘ └────────────────┘ └──────────────────┘  │
│                                                                                   │
│  ┌────────────────┐ ┌────────────────┐                                           │
│  │ Integration BC │ │ Report BC      │                                           │
│  │ (支撑域)        │ │ (支撑域)        │                                           │
│  │                │ │                │                                           │
│  │ WebhookConfig  │ │ AgentReport    │                                           │
│  │ WebhookDelivery│ │ SkillGrpReport │                                           │
│  │ ScreenPopConfig│ │ GroupAgentRpt  │                                           │
│  │ SessionInfoTmpl│ │ AgentStatusLog │                                           │
│  │ CSATConfig     │ │ Dashboard      │                                           │
│  │ CSATResult     │ │                │                                           │
│  │ DNCList        │ │                │                                           │
│  └────────────────┘ └────────────────┘                                           │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## 全局分层架构

```
cmd/server/main.go
│
├── internal/
│   ├── domain/                    ← 领域层（纯业务逻辑，零外部依赖）
│   │   ├── identity/              ← Phase 0
│   │   ├── call/                  ← Phase 1-3
│   │   ├── routing/               ← Phase 1
│   │   ├── telephony/             ← Phase 1-2
│   │   ├── campaign/              ← Phase 6
│   │   ├── crm/                   ← Phase 7
│   │   ├── ticket/                ← Phase 7
│   │   ├── im/                    ← Phase 8
│   │   ├── ai/                    ← Phase 9
│   │   ├── configuration/         ← Phase 0
│   │   ├── operation/             ← Phase 0
│   │   ├── integration/           ← Phase 3-4
│   │   ├── report/                ← Phase 4
│   │   └── platform/              ← Phase 0
│   │
│   ├── application/               ← 应用层（用例编排，事务管理）
│   ├── infrastructure/            ← 基础设施层（MySQL/Redis/FreeSWITCH/Kafka/NATS/MinIO 适配器）
│   └── interfaces/                ← 接口层（HTTP Handler + WebSocket + 路由）
│
├── migrations/                    ← SQL schema migrations
├── pkg/                           ← 可复用公共包
└── docs/                          ← 文档
```

---

## 全局技术选型

| 用途 | 库 | Phase |
|---|---|---|
| HTTP Router | `github.com/go-chi/chi/v5` | 0 |
| MySQL Driver | `github.com/go-sql-driver/mysql` | 0 |
| DB Access | `github.com/jmoiron/sqlx` | 0 |
| Migration | `github.com/golang-migrate/migrate/v4` | 0 |
| Snowflake ID | `github.com/bwmarrin/snowflake` | 0 |
| Redis | `github.com/redis/go-redis/v9` | 0 |
| JWT | `github.com/golang-jwt/jwt/v5` | 0 |
| JWKS | `github.com/MicahParks/keyfunc/v3` | 0 |
| Validation | `github.com/go-playground/validator/v10` | 0 |
| Testing | `testing` + `github.com/stretchr/testify` | 0 |
| Logging | `github.com/rs/zerolog` (结构化 JSON 日志) | 0 |
| Metrics | `github.com/prometheus/client_golang` | 0 |
| FreeSWITCH ESL | `github.com/percipia/eslgo` | 1 |
| NATS | `github.com/nats-io/nats.go` | 1 |
| Kafka | `github.com/segmentio/kafka-go` | 1 |
| MinIO | `github.com/minio/minio-go/v7` | 1 |
| WebSocket | `github.com/gorilla/websocket` | 1 |
| Aliyun SMS | `github.com/alibabacloud-go/dysmsapi-20170525/v4` | 3 |
| Aliyun ASR/TTS | `github.com/alibabacloud-go/nls` (或 REST) | 9 |

---

## 高并发架构设计 (1000+ 并发)

### 连接池规范

| 组件 | 参数 | 值 | Phase |
|---|---|---|---|
| MySQL | max_open_conns / max_idle_conns / conn_max_lifetime | 50 / 25 / 5min | 0 |
| Redis | pool_size / min_idle_conns | 100 / 20 | 0 |
| ESL | 连接池大小 / 自动重连 / 熔断阈值 | 3-5 / 指数退避(max 30s) / 3次失败熔断30s | 1 |
| HTTP Server | ReadTimeout / WriteTimeout / MaxConns | 30s / 60s / 10000 | 0 |

### ESL 高可用策略

```
Go App → ESL Connection Pool (per FreeSWITCH 实例)
       → ESL Router (call_id → FS 实例映射, Redis 存储)
       → Circuit Breaker (3次失败 → 熔断30s → half-open探测)
       → Auto-reconnect (指数退避, max 30s)
```

### 写入热点缓解

| 表 | 峰值写入 | 策略 |
|---|---|---|
| call_events | ~2000/s | Kafka 异步 → 批量 INSERT (每 500ms / 100条) |
| ivr_tracking | ~500/s | Kafka 异步 → 批量 INSERT |
| agent_presence_log | ~100/s | 直写 MySQL (可承受) |
| queue_snapshots | ~50/s | 直写 + 7天保留清理 |

### WebSocket 跨实例方案

- 坐席/管理员 WebSocket 连接到任意 Go 实例
- 状态变更通过 NATS 广播到所有 Go 实例
- 各实例本地维护 WebSocket 连接池，收到 NATS 消息后推送给本实例连接的客户端

### 租户资源隔离

- per-tenant 并发通话限额: `tenant_settings.max_concurrent_calls`
- per-tenant API 限流: `tenant_settings.api_rate_limit_per_sec`
- Go 层在 ESL 发起呼叫前检查 Redis 中该租户当前活跃通话数

---

## Phase 0 — 基础设施

### 涉及限界上下文
- **Identity BC** (核心域) — Tenant, User, Agent, SkillGroup
- **Platform BC** (支撑域) — Auth/JWT, RateLimit, AuditLog
- **Configuration BC** (通用域) — BreakReason, DispositionCode, CustomFieldDef, CallTag
- **Operation BC** (通用域) — AudioFile, BusinessHours

### 聚合与领域事件

| 聚合根 | 实体 | 值对象 | 领域事件 |
|---|---|---|---|
| `Tenant` | TenantSettings | Code, Status | TenantCreated, TenantSuspended |
| `User` | Agent | Email, Phone, Role, WorkMode, Status | UserCreated, UserDisabled |
| `SkillGroup` | SkillGroupMember | RoutingPolicy, Level(1-10), Status | SkillGroupMemberChanged |
| `BreakReason` | — | Code, IsSystem | — |
| `DispositionCode` | — | Code, Category | — |
| `CustomFieldDefinition` | — | FieldType, Scope | — |
| `AudioFile` | — | Format, Category | — |
| `BusinessHours` | Schedule | TimeRange, DayOfWeek | — |
| `AuditLog` | — | Action, ResourceRef, IP | — |

### TDD 范围（核心域先写测试）

```
domain/identity/service_test.go:
  TestTenantService_Create_Success
  TestTenantService_Create_DuplicateCode_Error
  TestTenantService_Create_EmptyCode_Error
  TestTenantService_Get_NotFound_Error
  TestTenantService_Update_Success
  TestTenantService_Suspend_ActiveTenant
  TestTenantService_Suspend_AlreadySuspended_Error
  TestTenantService_UpdateSettings_Success
  TestTenantService_UpdateSettings_InvalidRetentionDays

  TestUserService_Create_Success
  TestUserService_Create_DuplicateUsername_Error
  TestUserService_Create_InvalidEmail_Error
  TestUserService_Create_InvalidRole_Error
  TestUserService_Update_Success
  TestUserService_Disable_Success
  TestUserService_Disable_AlreadyDisabled_Error
  TestUserService_List_WithPagination

  TestAgentService_Create_Success
  TestAgentService_Create_UserNotFound_Error
  TestAgentService_Update_MaxConcurrentBounds

  TestSkillGroupService_Create_Success
  TestSkillGroupService_Create_DuplicateCode_Error
  TestSkillGroupService_AddMember_Success
  TestSkillGroupService_AddMember_AlreadyExists_Error
  TestSkillGroupService_AddMember_LevelOutOfRange_Error
  TestSkillGroupService_RemoveMember_Success
  TestSkillGroupService_RemoveMember_NotFound_Error
  TestSkillGroupService_UpdateRoutingPolicy_InvalidPolicy_Error
```

### 任务分解

| # | 任务 | 类型 | 预计 |
|---|---|---|---|
| 0.1 | `go mod init` + 目录结构 + 依赖安装 | 骨架 | 15min |
| 0.2 | `migrations/000001_init_schema.up.sql` (69张表) | DB | 25min |
| 0.3 | Snowflake ID 生成器 | Infra | 5min |
| 0.4 | Domain: Identity BC entity + repository 接口 | Domain | 15min |
| 0.5 | Domain: Identity BC service_test.go (TDD 先写测试) | TDD | 30min |
| 0.6 | Domain: Identity BC service.go (使测试通过) | Domain | 30min |
| 0.7 | Domain: Configuration BC (BreakReason/DispositionCode/CustomFieldDef/CallTag) | Domain | 15min |
| 0.8 | Domain: Operation BC (AudioFile/BusinessHours) | Domain | 15min |
| 0.9 | Infrastructure: MySQL repos (Tenant/User/Agent/SkillGroup) | Infra | 30min |
| 0.10 | Infrastructure: MySQL repos (通用域) | Infra | 15min |
| 0.11 | Infrastructure: JWT/Keycloak auth | Infra | 15min |
| 0.12 | Infrastructure: Redis 令牌桶限流 | Infra | 10min |
| 0.13 | Application: 用例编排 (Tenant/User/SkillGroup) | App | 20min |
| 0.14 | Middleware: Auth + RateLimit + Audit + TenantScope | Interface | 20min |
| 0.15 | HTTP Handlers + Router (全部 Phase 0 API) | Interface | 40min |
| 0.16 | `cmd/server/main.go` (DI + 启动) | 骨架 | 10min |
| 0.17 | Infrastructure: Prometheus metrics endpoint (`/metrics`) | Infra | 10min |
| 0.18 | Infrastructure: 结构化日志 (zerolog, JSON format) | Infra | 10min |
| 0.19 | `go vet` + `go build` + `go test ./...` | 验证 | 10min |

### API 路由

```
POST   /api/v1/tenants                          # 创建租户
GET    /api/v1/tenants                          # 租户列表
GET    /api/v1/tenants/{id}                     # 租户详情
PUT    /api/v1/tenants/{id}                     # 更新租户
GET    /api/v1/tenants/{id}/settings            # 租户设置
PUT    /api/v1/tenants/{id}/settings            # 更新设置

POST   /api/v1/users                            # 创建用户
GET    /api/v1/users                            # 用户列表
GET    /api/v1/users/{id}                       # 用户详情
PUT    /api/v1/users/{id}                       # 更新用户
DELETE /api/v1/users/{id}                       # 删除用户(软删)
POST   /api/v1/users/{id}/agent                 # 创建坐席配置
GET    /api/v1/users/{id}/agent                 # 坐席配置详情
PUT    /api/v1/users/{id}/agent                 # 更新坐席配置

POST   /api/v1/skill-groups                     # 创建技能组
GET    /api/v1/skill-groups                     # 技能组列表
GET    /api/v1/skill-groups/{id}                # 技能组详情
PUT    /api/v1/skill-groups/{id}                # 更新技能组
DELETE /api/v1/skill-groups/{id}                # 删除技能组
GET    /api/v1/skill-groups/{id}/members        # 成员列表
POST   /api/v1/skill-groups/{id}/members        # 添加成员
DELETE /api/v1/skill-groups/{id}/members/{uid}  # 移除成员

POST   /api/v1/audio-files                      # 上传音频
GET    /api/v1/audio-files                      # 音频列表
GET    /api/v1/audio-files/{id}                 # 音频详情
DELETE /api/v1/audio-files/{id}                 # 删除音频

POST   /api/v1/business-hours                   # 创建营业时间
GET    /api/v1/business-hours                   # 营业时间列表
GET    /api/v1/business-hours/{id}              # 详情
PUT    /api/v1/business-hours/{id}              # 更新
DELETE /api/v1/business-hours/{id}              # 删除

POST   /api/v1/break-reasons                    # 创建小休原因
GET    /api/v1/break-reasons                    # 列表
PUT    /api/v1/break-reasons/{id}               # 更新
DELETE /api/v1/break-reasons/{id}               # 删除

POST   /api/v1/disposition-codes                # 创建结案代码
GET    /api/v1/disposition-codes                # 列表
PUT    /api/v1/disposition-codes/{id}           # 更新
DELETE /api/v1/disposition-codes/{id}           # 删除

POST   /api/v1/custom-fields                    # 创建自定义字段
GET    /api/v1/custom-fields                    # 列表(按scope筛选)
PUT    /api/v1/custom-fields/{id}               # 更新
DELETE /api/v1/custom-fields/{id}               # 删除

POST   /api/v1/call-tags                        # 创建标签
GET    /api/v1/call-tags                        # 列表
PUT    /api/v1/call-tags/{id}                   # 更新
DELETE /api/v1/call-tags/{id}                   # 删除

GET    /api/v1/audit-logs                       # 审计日志(只读)
```

### 前端页面 (Phase 0 对应)
- 平台管理控制台 → 概览（产品状态占位）
- 平台管理控制台 → 实例管理（租户 CRUD）
- 实例内 → 客服管理 → 坐席列表/CRUD
- 实例内 → 客服管理 → 技能组列表/CRUD/成员管理
- 实例内 → 流程管理 → 音频管理（上传/播放/分类）
- 实例内 → 设置 → 呼入控制（号码标签管理基础）
- 实例内 → 设置 → 小休原因管理（CRUD，含3个系统小休码只读显示）
- 实例内 → 设置 → 结案代码管理（CRUD/分类）
- 实例内 → 设置 → 营业时间管理（CRUD/每日时段/节假日配置）
- 实例内 → 设置 → 自定义字段管理（CRUD，scope: customer/session_info/ticket）

### 退出标准
- [ ] 核心域 TDD 测试全部通过 (`go test ./internal/domain/...`)
- [ ] 所有 API 路由可用（curl 验证）
- [ ] JWT 认证生效（无 token → 401）
- [ ] 多租户隔离（不同 tenant_id 数据隔离）
- [ ] RBAC 生效（agent 不可访问 admin 接口）
- [ ] 审计日志自动记录所有写操作
- [ ] API per-tenant 限流生效（超限 → 429）
- [ ] `go build` + `go vet` + `go test ./...` 全部通过
- [ ] 创建租户时自动初始化3个系统小休码

---

## Phase 1 — 呼入 MVP

### 新增限界上下文
- **Call BC** (核心域) — Call, CallEvent, IVRTracking, Recording, QueueSnapshot
- **Routing BC** (核心域) — IVRFlow, IVRFlowVersion (完整20种节点)
- **Telephony BC** (支撑域) — Carrier, SIPTrunk, PhoneNumber

### 新增聚合

| 聚合根 | 实体 | 领域事件 |
|---|---|---|
| `Call` | CallEvent, IVRTracking | CallCreated, CallAnswered, CallEnded |
| `Recording` | — | RecordingCreated |
| `IVRFlow` | IVRFlowVersion | FlowPublished, FlowLocked, FlowCloned |
| `Carrier` | — | — |
| `SIPTrunk` | — | TrunkHealthChanged |
| `PhoneNumber` | SkillGroupBinding, DedicatedAgentBinding | — |
| `AutoTagRule` | — | — |
| `CallNumberTag` | — | — |

### TDD 范围

```
domain/routing/service_test.go:
  TestIVRFlowService_Create_Success
  TestIVRFlowService_Publish_DraftToPublished
  TestIVRFlowService_Publish_InvalidGraph_Error (未连接出口)
  TestIVRFlowService_Lock_Success
  TestIVRFlowService_Lock_AlreadyLocked_Error
  TestIVRFlowService_Unlock_NotOwner_Error
  TestIVRFlowService_Clone_Success
  TestIVRFlowService_Rollback_ToVersion
  TestIVRFlowService_ValidateNode_AllTypes (20种节点配置验证)

domain/call/service_test.go:
  TestCallService_CreateInboundCall
  TestCallService_RecordIVRTracking_NodeSequence
  TestCallService_CalculateDurations (ivr/ring/queue/wait)
  TestCallService_EndCall_WithHangupReason
```

### 任务分解

| # | 任务 | 类型 |
|---|---|---|
| 1.1 | Domain: Routing BC — IVRFlow entity + 20种节点验证 + 状态机 + 编辑锁 | Domain/TDD |
| 1.2 | Domain: Call BC — Call entity + CallEvent + IVRTracking | Domain/TDD |
| 1.3 | Domain: Telephony BC — Carrier/SIPTrunk/PhoneNumber entity | Domain |
| 1.4 | Domain: Configuration BC — AutoTagRule/CallNumberTag | Domain |
| 1.5 | Infrastructure: FreeSWITCH ESL 连接器 (`percipia/eslgo`) + 连接池(3-5) + 自动重连(指数退避) + 熔断器 | Infra |
| 1.6 | Infrastructure: PJSIP SIP Edge sidecar 集成 | Infra |
| 1.7 | Infrastructure: FreeSWITCH mod_verto WebRTC 配置 | Infra |
| 1.8 | Infrastructure: coturn TURN 服务器配置 | Infra |
| 1.9 | Infrastructure: NATS JetStream 连接 + 事件发布 | Infra |
| 1.10 | Infrastructure: Kafka 连接 + CDR 事件 | Infra |
| 1.11 | Infrastructure: MinIO 客户端 + 录音存储 | Infra |
| 1.12a | Application: IVR 引擎核心 — DAG 遍历器 + 变量作用域 + 节点分发 | App/核心 |
| 1.12b | Application: IVR 基础节点 — start/end/play/set_variable/branch/hangup_reason | App/核心 |
| 1.12c | Application: IVR 交互节点 — collect_dtmf/voicemail/satisfaction_rating | App/核心 |
| 1.12d | Application: IVR 路由节点 — transfer_to_agent/transfer_to_external/blind_transfer/callback | App/核心 |
| 1.12e | Application: IVR 集成节点 — function/http_request/json_parser/sms | App/核心 |
| 1.12f | Application: IVR 高级节点 — sub_flow/digital_employee/asr(Phase 9 桩) | App/核心 |
| 1.13 | Application: 录音服务（FreeSWITCH uuid_record → 本地 → MinIO） | App |
| 1.14 | Application: 号码标签/呼入控制/自动打标引擎 | App |
| 1.15 | HTTP Handlers: IVR流程CRUD/发布/克隆/导入导出/版本管理 | Interface |
| 1.16 | HTTP Handlers: Carrier/SIPTrunk/PhoneNumber CRUD | Interface |
| 1.17 | WebSocket: mod_verto 信令代理（坐席浏览器 ↔ FreeSWITCH） | Interface |
| 1.18 | 前端: 坐席工作台基础（接听/挂机按钮） | Frontend |
| 1.19 | 前端: IVR 画布编辑器（20种节点拖拽+连线） | Frontend |
| 1.20 | 前端: 号码管理页面 | Frontend |

### API 路由 (新增)

```
# IVR 流程
POST   /api/v1/ivr-flows
GET    /api/v1/ivr-flows
GET    /api/v1/ivr-flows/{id}
PUT    /api/v1/ivr-flows/{id}
POST   /api/v1/ivr-flows/{id}/publish
POST   /api/v1/ivr-flows/{id}/lock
POST   /api/v1/ivr-flows/{id}/unlock
POST   /api/v1/ivr-flows/{id}/clone
POST   /api/v1/ivr-flows/{id}/export
POST   /api/v1/ivr-flows/import
GET    /api/v1/ivr-flows/{id}/versions
POST   /api/v1/ivr-flows/{id}/rollback/{version}

# Telephony
POST   /api/v1/carriers
GET    /api/v1/carriers
POST   /api/v1/sip-trunks
GET    /api/v1/sip-trunks
GET    /api/v1/sip-trunks/{id}
PUT    /api/v1/sip-trunks/{id}
POST   /api/v1/phone-numbers
GET    /api/v1/phone-numbers
GET    /api/v1/phone-numbers/{id}
PUT    /api/v1/phone-numbers/{id}

# 号码标签
POST   /api/v1/call-number-tags
GET    /api/v1/call-number-tags
DELETE /api/v1/call-number-tags/{id}
POST   /api/v1/auto-tag-rules
GET    /api/v1/auto-tag-rules
PUT    /api/v1/auto-tag-rules/{id}
DELETE /api/v1/auto-tag-rules/{id}

# 录音
GET    /api/v1/recordings
GET    /api/v1/recordings/{id}
GET    /api/v1/recordings/{id}/stream          # 在线播放
GET    /api/v1/recordings/{id}/download        # 下载

# 语音信箱
GET    /api/v1/voicemails                       # 语音信箱列表
GET    /api/v1/voicemails/{id}                  # 详情
PATCH  /api/v1/voicemails/{id}/read             # 标记已读
DELETE /api/v1/voicemails/{id}                  # 删除
```

### 前端页面
- 实例内 → 流程管理 → IVR 画布编辑器（20种节点拖拽/连线/配置面板/保存/发布/克隆/导入导出/历史版本）
- 实例内 → 号码管理（列表/绑定IVR/用途设置/分组）
- 实例内 → 设置 → 呼入控制（号码标签手动添加/文件上传/自动打标规则）
- 实例内 → 话务报表 → 语音信箱（列表/播放/标记已读/删除/筛选）
- 坐席工作台基础（接听/挂机/通话状态显示）

### 退出标准
- [ ] 真实 PSTN 来电经 IVR 路由到坐席并被接听
- [ ] IVR 20种节点全部可执行
- [ ] 录音自动生成，可在线播放和下载
- [ ] 远程坐席可通过 WebRTC 接听（TURN 穿透）
- [ ] IVR 轨迹可查询（每个节点的进入/离开时间/变量/出口）
- [ ] IVR 流程支持编辑锁/克隆/导入导出/版本回滚
- [ ] 号码标签和自动打标规则生效
- [ ] IVR 语音信箱节点可录制留言，前端可播放/标记已读
- [ ] 单通话端到端延迟 < 500ms (IVR 接通到媒体播放)
- [ ] 10 并发通话无异常

---

## Phase 2 — 呼出 MVP

### 涉及限界上下文
- **Call BC** — 外呼通话
- **Telephony BC** — RoutingRule, CLIPolicy
- **Integration BC** — DNCList

### 新增聚合

| 聚合根 | 领域事件 |
|---|---|
| `RoutingRule` | — |
| `CLIPolicy` | — |
| `DNCList` | DNCEntryAdded |
| `CallTagAssignment` | — |

### TDD 范围

```
domain/telephony/service_test.go:
  TestRoutingService_MatchRule_CLIPrefix
  TestRoutingService_MatchRule_TimeOfDay
  TestRoutingService_MatchRule_Priority
  TestCLIPolicyService_SelectCLI_Strategy

domain/call/service_test.go:
  TestCallService_CreateOutboundCall
  TestCallService_CreateOutboundCall_DNCBlocked
  TestCallService_CreateInternalCall
  TestCallService_AssignTag
```

### 任务分解

| # | 任务 | 类型 |
|---|---|---|
| 2.1 | Domain: Telephony BC — RoutingRule 匹配引擎 + CLIPolicy | Domain/TDD |
| 2.2 | Domain: Integration BC — DNCList entity + 检查逻辑 | Domain/TDD |
| 2.3 | Domain: Call BC — 外呼流程 + 内部通话 + CallTagAssignment | Domain |
| 2.4 | Application: 外呼用例（DNC检查 → 路由匹配 → CLI选号 → ESL originate） | App |
| 2.5 | Application: 通话记录查询（含IVR/振铃/排队/等待时长） | App |
| 2.6 | HTTP Handlers: RoutingRule/CLIPolicy/DNCList CRUD | Interface |
| 2.7 | HTTP Handlers: 通话记录列表/详情/IVR轨迹/录音播放 | Interface |
| 2.8 | 前端: 通话记录页面（列表/筛选/详情弹窗/IVR轨迹/录音播放器） | Frontend |

### API 路由 (新增)

```
# Trunk 路由规则
POST   /api/v1/routing-rules
GET    /api/v1/routing-rules
PUT    /api/v1/routing-rules/{id}
DELETE /api/v1/routing-rules/{id}

# CLI 选号策略
POST   /api/v1/cli-policies
GET    /api/v1/cli-policies
PUT    /api/v1/cli-policies/{id}

# DNC
POST   /api/v1/dnc-list
GET    /api/v1/dnc-list
DELETE /api/v1/dnc-list/{id}
POST   /api/v1/dnc-list/check                  # 批量检查

# 通话记录
GET    /api/v1/calls
GET    /api/v1/calls/{id}
GET    /api/v1/calls/{id}/events
GET    /api/v1/calls/{id}/ivr-tracking
GET    /api/v1/calls/export                     # 导出

# 通话标签
POST   /api/v1/calls/{id}/tags
DELETE /api/v1/calls/{id}/tags/{tagId}

# 外呼
POST   /api/v1/calls/dial                       # 坐席外呼
POST   /api/v1/calls/internal-dial              # 坐席间通话
```

### 前端页面
- 话务报表 → 通话记录（列表/10000条限制/筛选/详情弹窗/IVR轨迹/录音播放器/下载）
- 话务报表 → 内部通话
- 坐席工作台 → 外呼按钮 + 拨号盘

### 退出标准
- [ ] 坐席可外呼并选择正确 CLI
- [ ] DNC 号码被拦截
- [ ] 坐席间可互呼
- [ ] 通话记录含完整时长字段 (IVR/振铃/排队/等待)
- [ ] 通话标签可添加/删除

---

## Phase 3 — 工作台 v1

### 涉及限界上下文
- **Call BC** — Hold/Transfer/Conference/Callback/EWT
- **Identity BC** — AgentPresence 增强 (DIALING + 子状态 + 工作模式)
- **Integration BC** — WebhookConfig, ScreenPopConfig

### 新增聚合

| 聚合根 | 领域事件 |
|---|---|
| `WebhookConfig` | — |
| `WebhookDeliveryLog` | — |
| `ScreenPopConfig` | — |
| `CallbackRequest` | CallbackRequested, CallbackExecuted |

### TDD 范围

```
domain/call/service_test.go:
  TestCallService_HoldCall
  TestCallService_RetrieveCall
  TestCallService_BlindTransfer_ToSkillGroup
  TestCallService_BlindTransfer_ToAgent
  TestCallService_BlindTransfer_ToExternal
  TestCallService_SendDTMF
  TestCallService_RequestCallback
  TestCallService_ExecuteCallback

domain/identity/service_test.go:
  TestAgentPresence_StateTransition_DialingToTalking
  TestAgentPresence_SubState_Monitored
  TestAgentPresence_WorkMode_Switch
  TestAgentPresence_ACW_WithDispositionCode
```

### 任务分解

| # | 任务 | 类型 |
|---|---|---|
| 3.1 | Domain: Call BC — Hold/Mute/Transfer(cold+blind)/Conference 状态机 | Domain/TDD |
| 3.2 | Domain: Call BC — CallbackRequest + EWT 计算 | Domain/TDD |
| 3.3 | Domain: Identity BC — AgentPresence 增强 (DIALING/sub_state/work_mode) | Domain/TDD |
| 3.4 | Domain: Integration BC — WebhookConfig/ScreenPopConfig | Domain |
| 3.5 | Application: ESL 通话控制（uuid_hold/uuid_bridge/uuid_transfer/uuid_send_dtmf） | App |
| 3.6 | Application: Webhook 投递引擎（含重试/签名/日志） | App |
| 3.7 | Application: 排队回呼调度器（坐席空闲时自动外呼） | App |
| 3.8 | Application: EWT 计算引擎（Redis 滑动窗口） | App |
| 3.9 | Application: 来电弹屏 URL 拼接 + iframe 参数传递 | App |
| 3.10 | HTTP Handlers: WebhookConfig/ScreenPopConfig CRUD | Interface |
| 3.11 | 前端: 坐席工作台完整（软电话条状态机 + 所有按钮 + 工作模式切换） | Frontend |
| 3.12 | 前端: 来电弹屏 (iframe + 多标签) | Frontend |
| 3.13 | 前端: 转接弹窗（技能组/坐席/外线 + 盲转/咨询转） | Frontend |
| 3.14 | 前端: 设置 → 来电弹屏配置页 | Frontend |
| 3.15 | 前端: 设置 → 事件推送配置页 | Frontend |
| 3.16 | 前端: 设置 → 短信配置页（签名/模板/渠道） | Frontend |
| 3.17 | Application: 短信发送服务（阿里云 SMS 集成） | App |
| 3.18 | Application: 快捷回复 CRUD（按全局/技能组/坐席维度） | App |
| 3.19 | HTTP Handlers: 快捷回复 API | Interface |
| 3.20 | 前端: 设置 → 快捷回复管理页 | Frontend |
| 3.21 | 前端: 坐席工作台 → 快捷回复选择面板 | Frontend |

### API 路由 (新增)

```
# 通话控制
POST   /api/v1/calls/{id}/hold
POST   /api/v1/calls/{id}/retrieve
POST   /api/v1/calls/{id}/mute
POST   /api/v1/calls/{id}/unmute
POST   /api/v1/calls/{id}/blind-transfer
POST   /api/v1/calls/{id}/send-dtmf
POST   /api/v1/calls/{id}/end

# 坐席状态
POST   /api/v1/agent/check-in
POST   /api/v1/agent/check-out
POST   /api/v1/agent/ready
POST   /api/v1/agent/break
POST   /api/v1/agent/acw
PUT    /api/v1/agent/work-mode

# 回呼
POST   /api/v1/callback-requests
GET    /api/v1/callback-requests

# Webhook
POST   /api/v1/webhook-configs
GET    /api/v1/webhook-configs
PUT    /api/v1/webhook-configs/{id}
DELETE /api/v1/webhook-configs/{id}

# 来电弹屏
POST   /api/v1/screen-pop-configs
GET    /api/v1/screen-pop-configs
PUT    /api/v1/screen-pop-configs/{id}
DELETE /api/v1/screen-pop-configs/{id}

# 短信配置
POST   /api/v1/sms-configs                      # 创建短信配置
GET    /api/v1/sms-configs                      # 列表
PUT    /api/v1/sms-configs/{id}                 # 更新
DELETE /api/v1/sms-configs/{id}                 # 删除
POST   /api/v1/sms/send                         # 按需发送短信

# 快捷回复
POST   /api/v1/quick-replies
GET    /api/v1/quick-replies
PUT    /api/v1/quick-replies/{id}
DELETE /api/v1/quick-replies/{id}
GET    /api/v1/quick-replies/available              # 坐席获取可用快捷回复
```

### 退出标准
- [ ] 坐席可 Hold/Mute/Transfer(冷转+盲转)/二次拨号
- [ ] ACW 含结案代码
- [ ] 坐席状态含 DIALING + 子状态 + 工作模式
- [ ] 来电弹屏可配置（最多5个 iframe, HTTPS）
- [ ] 事件推送 Webhook 可配置并投递
- [ ] 排队回呼生效（EWT 播报 + 自动回呼）
- [ ] 工作模式可切换（场内/场外/办公电话）
- [ ] 短信配置可管理（签名/模板/渠道），可手动/API 发送短信
- [ ] 快捷回复可管理（全局/技能组/坐席维度），坐席工作台可选用

---

## Phase 4 — 实时监控 + 报表

### 涉及限界上下文
- **Report BC** (新增，支撑域) — Dashboard, AgentReport, SkillGroupReport, GroupAgentReport, AgentStatusLog
- **Integration BC** — CSATConfig, CSATResult

### 新增聚合

| 聚合根 | 说明 |
|---|---|
| `Dashboard` | 实时聚合指标（Redis HASH + WebSocket 推送） |
| `AgentReport` | 坐席报表（30+字段，5维度，MySQL 聚合查询） |
| `SkillGroupReport` | 技能组报表 |
| `GroupAgentReport` | 分组坐席报表（按技能组维度） |
| `AgentStatusLog` | 坐席状态日志查询 |
| `CSATConfig` | 满意度调查配置 |
| `CSATResult` | 满意度评价结果 |

### TDD 范围

```
domain/report/service_test.go:
  TestDashboard_CalculateServiceLevel20s
  TestDashboard_CalculateAgentUtilization
  TestDashboard_CallFunnel_Ratios
  TestAgentReport_Aggregate_30Fields
  TestGroupAgentReport_BySkillGroup
```

### 任务分解

| # | 任务 | 类型 |
|---|---|---|
| 4.1 | Domain: Report BC — Dashboard 聚合指标定义 + 服务水平计算 | Domain/TDD |
| 4.2 | Domain: Report BC — AgentReport 30+字段聚合查询 | Domain |
| 4.3 | Domain: Report BC — GroupAgentReport (坐席×技能组) | Domain |
| 4.4 | Domain: Report BC — AgentStatusLog 查询 | Domain |
| 4.5 | Domain: Integration BC — CSATConfig/CSATResult | Domain |
| 4.6 | Infrastructure: Redis 实时指标聚合 (HASH/ZSET) | Infra |
| 4.7 | Application: Dashboard WebSocket 推送（5s刷新） | App |
| 4.8 | Application: 报表导出（CSV/Excel，大数据量分批） | App |
| 4.9 | Application: 满意度调查触发（IVR节点 + SMS） | App |
| 4.10 | HTTP Handlers: Dashboard/Report/CSAT API | Interface |
| 4.11 | 前端: 概览页（7大指标区块+漏斗图+AI服务概览+大屏模式） | Frontend |
| 4.12 | 前端: 坐席报表（30+字段/自定义表头/5维度切换/导出） | Frontend |
| 4.13 | 前端: 分组坐席报表 | Frontend |
| 4.14 | 前端: 技能组报表 | Frontend |
| 4.15 | 前端: 坐席状态日志（小休码筛选） | Frontend |
| 4.16 | 前端: 设置 → 满意度调研配置页 | Frontend |
| 4.17 | **前端: 双呼报表独立 Tab（双呼通话量/接通率/时长/平均时长）** | Frontend |
| 4.18 | **前端: 内部呼叫报表独立 Tab（内呼发起量/接通率/时长）** | Frontend |
| 4.19 | **前端: 技能组报表增强（排队总量/排队放弃量/振铃放弃量/20s应答率）** | Frontend |
| 4.20 | **前端: 概览页漏斗图细化（IVR→机器人/人工 → 全服务/半服务/直转 → 实际接听）** | Frontend |

### API 路由 (新增)

```
# 实时监控
GET    /api/v1/dashboard/overview               # 概览指标
GET    /api/v1/dashboard/agent-status           # 坐席状态列表
GET    /api/v1/dashboard/skill-group-status     # 技能组状态
GET    /api/v1/dashboard/call-trend             # 话务趋势
GET    /api/v1/dashboard/call-funnel            # 呼入漏斗
WS     /api/v1/dashboard/ws                     # WebSocket实时推送

# 报表
GET    /api/v1/reports/agent                    # 坐席报表
GET    /api/v1/reports/agent/export             # 导出
GET    /api/v1/reports/group-agent              # 分组坐席报表
GET    /api/v1/reports/group-agent/export
GET    /api/v1/reports/skill-group              # 技能组报表(含排队总量/排队放弃/振铃放弃/20s应答率)
GET    /api/v1/reports/skill-group/export
GET    /api/v1/reports/back2back                 # 双呼报表
GET    /api/v1/reports/back2back/export
GET    /api/v1/reports/internal-call              # 内部呼叫报表
GET    /api/v1/reports/internal-call/export
GET    /api/v1/reports/agent-status-log         # 坐席状态日志
GET    /api/v1/reports/agent-status-log/export

# 满意度
POST   /api/v1/csat-configs
GET    /api/v1/csat-configs
PUT    /api/v1/csat-configs/{id}
GET    /api/v1/csat-results
```

### 退出标准
- [ ] 概览大屏含7大指标区块 + 漏斗图 + 大屏模式
- [ ] 坐席报表含完整30+字段，可自定义表头
- [ ] 分组坐席报表按技能组维度
- [ ] 20s 服务水平指标正确
- [ ] 报表可导出（大数据量分批）
- [ ] 满意度调查(IVR+SMS)可配置并收集
- [ ] 双呼报表独立 Tab 可查看
- [ ] 内部呼叫报表独立 Tab 可查看
- [ ] 技能组报表含排队总量/排队放弃量/振铃放弃量/20s应答率
- [ ] 概览页漏斗图细化到 IVR→机器人/人工→全服务/半服务/直转→实际接听

---

## Phase 5 — 工作台 v2 + 管理员

### 涉及限界上下文
- **Call BC** — 热转(咨询转接)/会议/监听/耳语/强插/强拆/**通话辅导(CoachCall)**
- **Identity BC** — 个人数据/设置

### TDD 范围

```
domain/call/service_test.go:
  TestCallService_AttendedTransfer
  TestCallService_ConsultTransfer_Initiate
  TestCallService_ConsultTransfer_Complete
  TestCallService_ConsultTransfer_Cancel
  TestCallService_Conference_ThreeWay
  TestCallService_Monitor_Listen
  TestCallService_Monitor_Whisper
  TestCallService_Monitor_Barge
  TestCallService_Monitor_Intercept
  TestCallService_Coach_Success
  TestCallService_Coach_Timeout
  TestCallService_Whisper_PreConnect
```

### 任务分解

| # | 任务 | 类型 |
|---|---|---|
| 5.1 | Domain: Call BC — 热转/咨询转接状态机 (AttendedTransfer + Consult → Transfer/Cancel) | Domain/TDD |
| 5.2 | Domain: Call BC — 会议/三方通话 (mod_conference) | Domain/TDD |
| 5.3 | Domain: Call BC — 监听/耳语/强插/强拆 (eavesdrop) | Domain/TDD |
| 5.3b | Domain: Call BC — **通话辅导 (CoachCall, 坐席听到辅导者客户听不到, 超时可配 默认30s)** | Domain/TDD |
| 5.4 | Domain: Call BC — 坐席耳语 (接通前播报) | Domain |
| 5.5 | Application: ESL conference/eavesdrop 命令封装 | App |
| 5.6 | Application: SIP 话机注册 (mod_sofia) | App |
| 5.7 | Application: 场外模式（桥接手机号） | App |
| 5.8 | HTTP Handlers: 会议/监听/转接 API | Interface |
| 5.9 | HTTP Handlers: 个人数据概览/个人设置 API | Interface |
| 5.10 | 前端: 工作台高级功能（咨询转/会议/监听按钮） | Frontend |
| 5.11 | 前端: 管理员监控面板（监听/耳语/强插/强拆按钮） | Frontend |
| 5.12 | 前端: 我的工作台 → 数据概览（个人统计） | Frontend |
| 5.13 | 前端: 我的工作台 → 设置（头像/密码/昵称/状态重置） | Frontend |

### API 路由 (新增)

```
# 高级通话控制
POST   /api/v1/calls/{id}/attended-transfer     # 热转（等目标接听后转）
POST   /api/v1/calls/{id}/consult               # 发起咨询
POST   /api/v1/calls/{id}/consult-transfer       # 咨询转接
POST   /api/v1/calls/{id}/consult-cancel         # 取消咨询
POST   /api/v1/calls/{id}/conference             # 三方通话
POST   /api/v1/calls/{id}/monitor                # 监听
POST   /api/v1/calls/{id}/whisper                # 耳语
POST   /api/v1/calls/{id}/barge                  # 强插
POST   /api/v1/calls/{id}/intercept              # 强拆
POST   /api/v1/calls/{id}/coach                  # 辅导（坐席听到辅导者，客户听不到）

# 个人
GET    /api/v1/me/overview                       # 个人统计
PUT    /api/v1/me/profile                        # 更新个人信息
PUT    /api/v1/me/password                       # 修改密码
POST   /api/v1/me/reset-state                    # 状态重置
```

### 退出标准
- [ ] 管理员可监听/耳语/强插/强拆/辅导坐席通话
- [ ] 辅导时坐席可听到辅导者，客户听不到
- [ ] 咨询转接（发起→完成/取消）流程正常
- [ ] 三方通话/会议正常
- [ ] 接通前坐席耳语播报来电信息
- [ ] SIP 话机可注册并接听
- [ ] 场外模式桥接手机号
- [ ] 坐席可查看个人统计 + 修改个人设置

---

## Phase 6 — Trunk v2 + 外呼进阶

### 涉及限界上下文
- **Telephony BC** — 多 Trunk HA/Failover
- **Campaign BC** (新增，核心域) — Campaign, CampaignCase

### 4 种外呼模式行为规范

| 模式 | 行为 | 核心参数 |
|---|---|---|
| **Predictive** | 系统根据坐席可用预测自动批量拨号。`ratio_multiplier` 控制每空闲坐席拨出比率。实时监控放弃率，超 `max_abandon_rate` 自动降速。| ratio_multiplier, max_abandon_rate |
| **Preview** | 系统推送案例到坐席屏幕，坐席查看客户资料后手动拨号/跳过。超 `preview_timeout_sec` 未操作自动跳过。| preview_timeout_sec |
| **Progressive** | 坐席空闲时系统自动拨出 1 通（1:1 固定比率），接通后自动桥接。| — |
| **Power** | 每个空闲坐席同时拨多通（固定 ratio），第一个接通的桥接给坐席，其余挂断。| ratio_multiplier |

### 新增聚合

| 聚合根 | 实体 | 领域事件 |
|---|---|---|
| `Campaign` | CampaignCase | CampaignStarted, CampaignCompleted |

### TDD 范围

```
domain/campaign/service_test.go:
  TestCampaignService_Create_Success
  TestCampaignService_Start_WithValidCases
  TestCampaignService_Pause_Running
  TestCampaignService_ImportCases_DNCFilter
  TestCampaignService_DialingMode_Predictive
  TestCampaignService_DialingMode_Preview
  TestCampaignService_DialingMode_Progressive
  TestCampaignService_DialingMode_Power
  TestCampaignService_RetryLogic

domain/telephony/service_test.go:
  TestTrunkHealthCheck_OPTIONSKeepalive
  TestTrunkFailover_AutoSwitch
```

### 任务分解

| # | 任务 | 类型 |
|---|---|---|
| 6.1 | Domain: Campaign BC — Campaign entity + 4种外呼模式状态机 | Domain/TDD |
| 6.2 | Domain: Telephony BC — Trunk 健康检查 + Failover 逻辑 | Domain/TDD |
| 6.3a | Application: Predictive Dialer — 坐席可用预测引擎 (滑动窗口算法) | App |
| 6.3b | Application: Predictive Dialer — 实时放弃率监控 + ratio 自适应调节 | App |
| 6.3c | Application: Predictive Dialer — 合规限速 (max_abandon_rate 触发降速) | App |
| 6.4a | Application: Preview 外呼 — 案例推送 + 预览超时 + 坐席接受/跳过 | App |
| 6.4b | Application: Progressive 外呼 — 1:1 坐席空闲自动拨号 | App |
| 6.4c | Application: Power 外呼 — 固定高比率并发拨号 | App |
| 6.5 | Application: Trunk OPTIONS keepalive + 自动切换 | App |
| 6.6 | Application: 双呼 B2B (背靠背通话) | App |
| 6.7 | Application: 闪信发送 | App |
| 6.8 | Application: 加密通话（隐私保护） | App |
| 6.8b | **Application: 号码脱敏外呼 (masked_callee, 坐席看到脱敏号码实际拨号使用真实号码)** | App |
| 6.9 | HTTP Handlers: Campaign CRUD/Start/Pause/Cases导入 | Interface |
| 6.10a | 前端: 批量外呼管理页（活动CRUD/案例导入/4种模式配置） | Frontend |
| 6.10b | 前端: 坐席工作台 — Preview 模式案例预览卡片 + 拨号/跳过操作 | Frontend |
| 6.10c | 前端: Campaign 实时大屏（并发数/接通率/放弃率/进度条/WebSocket） | Frontend |
| 6.11 | 前端: 话务报表 → 双呼记录 | Frontend |

### API 路由 (新增)

```
# Campaign
POST   /api/v1/campaigns
GET    /api/v1/campaigns
GET    /api/v1/campaigns/{id}
PUT    /api/v1/campaigns/{id}
POST   /api/v1/campaigns/{id}/start
POST   /api/v1/campaigns/{id}/pause
POST   /api/v1/campaigns/{id}/abort
POST   /api/v1/campaigns/{id}/cases/import
GET    /api/v1/campaigns/{id}/cases
GET    /api/v1/campaigns/{id}/stats              # 实时统计

# 双呼
POST   /api/v1/calls/back2back

# Campaign 实时推送
WS     /api/v1/ws/campaigns/{id}/live     # WebSocket 实时监控

# Trunk 管理增强
POST   /api/v1/sip-trunk-groups
GET    /api/v1/sip-trunk-groups
POST   /api/v1/sip-trunk-groups/{id}/members
```

### 退出标准
- [ ] 4种外呼模式 (Predictive/Preview/Progressive/Power) 均可运行
- [ ] Trunk 故障自动切换
- [ ] 双呼 B2B 正常
- [ ] Campaign 可导入案例/启动/暂停/中止
- [ ] 闪信可发送
- [ ] 加密通话（隐私保护）正常
- [ ] 号码脱敏外呼时坐席端显示脱敏号码，实际拨号使用真实号码

---

## Phase 7 — CRM + 工单 + 知识库

### 涉及限界上下文
- **CRM BC** (新增，核心域) — Customer, CustomerPhone, CustomerInteraction
- **Ticket BC** (新增，支撑域) — TicketCategory, TicketTemplate, Ticket, TicketComment
- **AI BC** (部分) — KnowledgeBase, AgentScript

### 新增聚合

| 聚合根 | 实体 | 领域事件 |
|---|---|---|
| `Customer` | CustomerPhone, CustomerInteraction | CustomerCreated |
| `TicketCategory` | — | — |
| `TicketTemplate` | — | TemplatePublished |
| `Ticket` | TicketComment | TicketCreated, TicketAssigned, TicketResolved |
| `KnowledgeCategory` | — | — |
| `KnowledgeArticle` | — | — |
| `AgentScript` | — | — |
| `SessionInfoTemplate` | — | — |

### TDD 范围

```
domain/crm/service_test.go:
  TestCustomerService_Create_Success
  TestCustomerService_Create_WithMultiPhones
  TestCustomerService_FindByPhone_AnyPhoneMatch
  TestCustomerService_BatchImport_CSV

domain/ticket/service_test.go:
  TestTicketService_Create_FromTemplate
  TestTicketService_Assign_Success
  TestTicketService_Transition_OpenToInProgress
  TestTicketService_Transition_InvalidState_Error
  TestTicketTemplateService_Publish_Online
  TestTicketTemplateService_FlowGraph_Validation
```

### 任务分解

| # | 任务 | 类型 |
|---|---|---|
| 7.1 | Domain: CRM BC — Customer entity + 多号码 + 自定义字段 | Domain/TDD |
| 7.2 | Domain: CRM BC — 批量导入/导出（CSV/Excel） | Domain |
| 7.3 | Domain: Ticket BC — 类目/模板/字段/流程节点/上线下线 | Domain/TDD |
| 7.4 | Domain: Ticket BC — 工单创建/指派/流转/完成 | Domain/TDD |
| 7.5 | Domain: AI BC — KnowledgeCategory/Article + 全文搜索 | Domain |
| 7.6 | Domain: AI BC — AgentScript (脚本内容 JSON) | Domain |
| 7.7 | Application: 来电弹屏完整（客户匹配 + 历史记录 + iframe 拼接） | App |
| 7.8 | Application: 自定义字段配置 UI 数据接口 | App |
| 7.9 | HTTP Handlers: Customer/Ticket/Knowledge/Script/SessionInfo CRUD | Interface |
| 7.10 | 前端: 业务管理 → 客户列表/CRUD/字段配置/批量导入导出 | Frontend |
| 7.11 | 前端: 业务管理 → 工单管理（字段/类目/模板画布/处理/流转） | Frontend |
| 7.12 | 前端: 业务管理 → 会话信息（字段管理/模板管理） | Frontend |
| 7.13 | 前端: 坐席工作台 → 知识库搜索面板 | Frontend |
| 7.14 | 前端: 坐席工作台 → 脚本引导面板 | Frontend |

### API 路由 (新增)

```
# 客户
POST   /api/v1/customers
GET    /api/v1/customers
GET    /api/v1/customers/{id}
PUT    /api/v1/customers/{id}
DELETE /api/v1/customers/{id}
POST   /api/v1/customers/import                  # 批量导入
GET    /api/v1/customers/export                  # 批量导出
GET    /api/v1/customers/by-phone/{phone}        # 按号码查

# 工单
POST   /api/v1/ticket-categories
GET    /api/v1/ticket-categories
POST   /api/v1/ticket-templates
GET    /api/v1/ticket-templates
PUT    /api/v1/ticket-templates/{id}
POST   /api/v1/ticket-templates/{id}/publish
POST   /api/v1/ticket-templates/{id}/offline
POST   /api/v1/tickets
GET    /api/v1/tickets
GET    /api/v1/tickets/{id}
PUT    /api/v1/tickets/{id}
POST   /api/v1/tickets/{id}/assign
POST   /api/v1/tickets/{id}/comments

# 知识库
POST   /api/v1/knowledge-categories
GET    /api/v1/knowledge-categories
POST   /api/v1/knowledge-articles
GET    /api/v1/knowledge-articles
GET    /api/v1/knowledge-articles/{id}
PUT    /api/v1/knowledge-articles/{id}
GET    /api/v1/knowledge-articles/search?q=...   # 全文搜索

# 坐席脚本
POST   /api/v1/agent-scripts
GET    /api/v1/agent-scripts
PUT    /api/v1/agent-scripts/{id}

# 会话信息模板
POST   /api/v1/session-info-templates
GET    /api/v1/session-info-templates
PUT    /api/v1/session-info-templates/{id}
```

### 退出标准
- [ ] 来电时弹出客户历史 + 工单
- [ ] 客户支持多号码 + 自定义字段 + 批量导入导出
- [ ] 工单可创建/指派/流转/完成，含类目/模板/流程节点
- [ ] 坐席可搜索知识库
- [ ] 坐席脚本引导在工作台可见
- [ ] 通话结束可创建工单

---

## Phase 8 — 全渠道 / 在线客服

### 涉及限界上下文
- **IM BC** (新增，支撑域) — IMChannel, IMSession, IMMessage

### 新增聚合

| 聚合根 | 实体 | 领域事件 |
|---|---|---|
| `IMChannel` | — | — |
| `IMSession` | IMMessage | SessionCreated, SessionAssigned, SessionEnded |

### TDD 范围

```
domain/im/service_test.go:
  TestIMService_CreateSession_RouteToSkillGroup
  TestIMService_AssignAgent_LongestIdle
  TestIMService_TransferSession
  TestIMService_CloseSession
  TestIMService_SendMessage_Text
  TestIMService_SendMessage_Image
  TestIMService_MaxChatSlots_Exceeded
```

### 任务分解

| # | 任务 | 类型 |
|---|---|---|
| 8.1 | Domain: IM BC — IMChannel/Session/Message entity + 路由 | Domain/TDD |
| 8.2 | Application: 在线会话路由（技能组分配 + 坐席并发限制） | App |
| 8.3 | Application: WebSocket 消息推送（坐席 ↔ 访客） | App |
| 8.4 | Application: Email 渠道接入（收件/发件解析） | App |
| 8.5 | HTTP Handlers: IM Channel/Session/Message API | Interface |
| 8.6 | 前端: 网页在线客服 Widget（嵌入式JS SDK） | Frontend |
| 8.7 | 前端: 坐席在线工作台（多会话标签切换/消息列表/发送） | Frontend |
| 8.8 | 前端: 网络业务 → 概览/报表/会话记录/人员/设置 | Frontend |
| 8.9 | 前端: 坐席工作台 → 在线聊天面板集成 | Frontend |
| 8.10 | Application: IM 坐席 AI 辅助 — AI 纠错/扩写/话术优化（LLM API 调用） | App |
| 8.11 | 前端: 在线工作台 → AI 辅助面板（输入框上方: 纠错/扩写/优化按钮） | Frontend |

### API 路由 (新增)

```
# IM 渠道
POST   /api/v1/im-channels
GET    /api/v1/im-channels
PUT    /api/v1/im-channels/{id}

# IM 会话
GET    /api/v1/im-sessions
GET    /api/v1/im-sessions/{id}
POST   /api/v1/im-sessions/{id}/transfer
POST   /api/v1/im-sessions/{id}/close

# IM 消息
GET    /api/v1/im-sessions/{id}/messages
POST   /api/v1/im-sessions/{id}/messages

# 访客端 (Widget)
POST   /api/v1/widget/sessions
POST   /api/v1/widget/sessions/{id}/messages
WS     /api/v1/widget/ws

# Email 渠道
POST   /api/v1/email/inbound                    # Email webhook 接收

# IM AI 辅助
POST   /api/v1/im/ai-assist/correct               # AI 纠错
POST   /api/v1/im/ai-assist/expand                # AI 扩写
POST   /api/v1/im/ai-assist/optimize              # 话术优化
```

### 退出标准
- [ ] 客户可通过网页 Widget 与坐席沟通
- [ ] 坐席可同时处理多个在线会话
- [ ] Email 渠道可收发
- [ ] 在线报表可查
- [ ] 会话路由分配到技能组/坐席
- [ ] IM 坐席 AI 辅助可用（纠错/扩写/优化）

---

## Phase 9 — AI + 智能化

### 涉及限界上下文
- **AI BC** — DigitalEmployee, DEScene, 智能质检(**实时+离线+大模型**), 坐席辅助(**实时转写+实时话术推荐+自动填单**), 热词分析, **会话标签分析**, AI摘要(**点击+自动**), 情绪分析, **AI满意度预测**, **ASR热词库**, **质检规则/方案/申诉**, **智能分析(IVR分析+完成度判断+话后动作)**
- **Report BC** — PerformanceScorecard

### 新增聚合

| 聚合根 | 领域事件 |
|---|---|
| `DigitalEmployee` | — |
| `DigitalEmployeeScene` | ScenePublished |
| `PerformanceScorecard` | — |
| `QARule` | — |
| `QAScheme` | — |
| `QAResult` | QACompleted, QAAppealed, QAReviewed |
| `ASRHotwords` | — |

### TDD 范围

```
domain/ai/service_test.go:
  TestDigitalEmployeeService_Create_Success
  TestDigitalEmployeeService_IntentMatch
  TestDigitalEmployeeService_TransferToHuman_Trigger
  TestQualityInspection_RuleMatch_Keyword
  TestQualityInspection_RuleMatch_Silence
  TestQualityInspection_RuleMatch_Speed
  TestQualityInspection_RuleMatch_LLM
  TestQualityInspection_SchemeScore_Calculation
  TestQualityInspection_Appeal_Flow
  TestRealtimeTranscription_StreamPush
  TestRealtimeAssist_ScriptRecommend
  TestAutoFormFill_ExtractFields
  TestSessionTagAnalysis_Classification
  TestAISatisfaction_Prediction
  TestSentimentAnalysis_Classification
  TestIVRAnalysis_PathSummary
  TestCompletionScore_Judgement
  TestPostCallActions_Extract
```

### 任务分解

| # | 任务 | 类型 |
|---|---|---|
| 9.1 | Domain: AI BC — DigitalEmployee/Scene entity + 意图/FAQ配置 | Domain/TDD |
| 9.2 | Infrastructure: 阿里通义 ASR Provider 实现 | Infra |
| 9.3 | Infrastructure: 阿里通义 TTS Provider 实现 | Infra |
| 9.4 | Application: 数字员工 IVR 集成（VOICE_NAVIGATOR 节点执行） | App |
| 9.5 | Application: 离线质检引擎（录音转文字 → 规则+LLM分析） | App |
| 9.6 | Application: 实时质检引擎（通话中实时转写 → 规则/LLM实时检测 → 告警推送） | App |
| 9.7 | Application: 质检规则管理（算子体系: 关键词/正则/相似度/静音/语速/抢话/能量/时长/实体/角色/异常挂机/LLM） | App |
| 9.8 | Application: 质检方案管理（多规则组合 + 行业预置模板） | App |
| 9.9 | Application: 质检申诉/复核流程 | App |
| 9.10 | Infrastructure: 实时语音转写服务（通话中 ASR 流式识别 → WebSocket 推送到坐席） | Infra |
| 9.11 | Application: 实时话术推荐（转写文本 → 知识库/脚本匹配 → 推送到坐席） | App |
| 9.12 | Application: 自动填单（通话/对话文本 → LLM 提取 → 工单字段自动填充） | App |
| 9.13 | Application: 会话标签分析（LLM 对通话内容自动分类标注） | App |
| 9.14 | Application: 热词分析（按时间段/技能组 AI 提取热词报表） | App |
| 9.15 | Application: AI 对话摘要（支持“坐席点击生成”和“后台自动生成”两种模式） | App |
| 9.16 | Application: 情绪分析（通话中实时 + 通话后离线） | App |
| 9.17 | Application: AI 满意度预测（无需客户参与，AI分析通话内容预测满意度） | App |
| 9.18 | Application: ASR 热词库管理（租户自定义热词改善识别） | App |
| 9.19 | Application: 绩效管理/记分卡 | App |
| 9.20 | HTTP Handlers: DigitalEmployee/Scene API | Interface |
| 9.21 | HTTP Handlers: 质检规则/方案/结果/申诉 API | Interface |
| 9.22 | HTTP Handlers: 实时转写/话术推荐/自动填单/会话标签/AI满意度/ASR热词库 API | Interface |
| 9.23 | HTTP Handlers: 绩效记分卡 API | Interface |
| 9.24 | 前端: 数字员工 → 场景管理/AI平台(意图/FAQ/对话流) | Frontend |
| 9.25 | 前端: 坐席工作台 → 实时转写面板 + 实时话术推荐 + 自动填单按钮 + AI摘要按钮 | Frontend |
| 9.26 | 前端: 数据监控 → 热词/会话标签分析 | Frontend |
| 9.27 | 前端: 质检管理 → 规则编辑/方案管理/结果列表/申诉管理 | Frontend |
| 9.28 | 前端: 设置 → 智能化设置/质检推送/ASR热词库管理 | Frontend |
| 9.29 | 前端: 通话记录 → AI 情绪/AI满意度/AI标签/AI摘要列 | Frontend |
| 9.30 | **Application: IVR 路径 AI 分析（坐席接起前自动分析用户 IVR 路径与关键信息）** | App |
| 9.31 | **Application: 完成度判断（大模型判断用户诉求是否解决 1-5 分）** | App |
| 9.32 | **Application: 话后处理动作自动抽取（AI 从通话内容提取待办事项清单）** | App |
| 9.33 | **前端: 通话详情 → 智能分析面板（IVR分析/完成度/话后动作）** | Frontend |

### API 路由 (新增)

```
# 数字员工
POST   /api/v1/digital-employees
GET    /api/v1/digital-employees
GET    /api/v1/digital-employees/{id}
PUT    /api/v1/digital-employees/{id}
POST   /api/v1/digital-employees/{id}/scenes
GET    /api/v1/digital-employees/{id}/scenes
PUT    /api/v1/digital-employees/{id}/scenes/{sid}
POST   /api/v1/digital-employees/{id}/test       # 测试对话

# 质检规则
POST   /api/v1/qa-rules
GET    /api/v1/qa-rules
GET    /api/v1/qa-rules/{id}
PUT    /api/v1/qa-rules/{id}
DELETE /api/v1/qa-rules/{id}

# 质检方案
POST   /api/v1/qa-schemes
GET    /api/v1/qa-schemes
GET    /api/v1/qa-schemes/{id}
PUT    /api/v1/qa-schemes/{id}
DELETE /api/v1/qa-schemes/{id}
POST   /api/v1/qa-schemes/{id}/rules              # 添加规则到方案
DELETE /api/v1/qa-schemes/{id}/rules/{ruleId}     # 移除

# 质检执行与结果
POST   /api/v1/quality-inspections/run             # 触发离线质检
GET    /api/v1/quality-inspections/results
GET    /api/v1/quality-inspections/results/{id}
POST   /api/v1/quality-inspections/results/{id}/appeal   # 申诉
POST   /api/v1/quality-inspections/results/{id}/review   # 复核

# 实时转写
WS     /api/v1/calls/{id}/realtime-transcript       # WebSocket 实时转写流

# 实时话术推荐
GET    /api/v1/calls/{id}/script-recommendations   # 获取当前推荐

# 自动填单
POST   /api/v1/calls/{id}/auto-fill-ticket         # AI 提取信息填充工单

# 会话标签
GET    /api/v1/calls/{id}/ai-tags                  # 获取 AI 标签
POST   /api/v1/session-tag-analysis/run            # 触发批量标签分析
GET    /api/v1/session-tag-analysis/results

# AI 摘要
POST   /api/v1/calls/{id}/ai-summary               # 坐席点击生成摘要

# AI 满意度预测
GET    /api/v1/calls/{id}/ai-satisfaction

# ASR 热词库
POST   /api/v1/asr-hotwords
GET    /api/v1/asr-hotwords
GET    /api/v1/asr-hotwords/{id}
PUT    /api/v1/asr-hotwords/{id}
DELETE /api/v1/asr-hotwords/{id}

# 绩效
GET    /api/v1/performance-scorecards
POST   /api/v1/performance-scorecards/generate     # 生成记分卡
```

### 退出标准
- [ ] AI 机器人可接听来电并按意图路由/转人工
- [ ] ASR/TTS 阿里通义集成可用
- [ ] 坐席通话中可看到实时语音转写
- [ ] 实时质检可检测违规并告警
- [ ] 实时话术推荐可推送到坐席工作台
- [ ] AI 自动填单可将对话信息填入工单
- [ ] 会话标签 AI 自动分类
- [ ] 大模型质检 + 规则质检并行运行
- [ ] 质检申诉/复核闭环
- [ ] ASR 热词库可配置
- [ ] AI 对话摘要支持“点击生成”和“自动生成”两种模式
- [ ] 情绪分析标注
- [ ] AI 满意度预测可运行
- [ ] 绩效记分卡可生成
- [ ] IVR 路径 AI 分析可在坐席接起前自动生成
- [ ] 完成度判断可正确评分 (1-5)
- [ ] 话后处理动作可自动抽取并展示

---

## Phase 10 — 规模化 + 加固

### 涉及限界上下文
- 所有 BC（横切关注点）

### 任务分解

| # | 任务 | 类型 |
|---|---|---|
| 10.1 | 多 FreeSWITCH 实例 + SIP LB (OpenSIPS/Kamailio) | Infra |
| 10.2 | HA failover 演练（MySQL 主从切换/Redis Cluster/FreeSWITCH） | Infra |
| 10.3a | 负载测试: 呼入压测 — 1000 并发呼入→IVR→ACD→接听→录音 | Test |
| 10.3b | 负载测试: 外呼压测 — Predictive Campaign 500 并发外呼 | Test |
| 10.3c | 负载测试: 混合压测 — 500 呼入 + 500 外呼同时运行 | Test |
| 10.3d | 负载测试: WebSocket 压测 — 500 坐席 + 10 管理员大屏同时在线 | Test |
| 10.3e | 负载测试: 报表压测 — 100 万条 CDR 查询 < 5s | Test |
| 10.4 | TLS + SRTP on trunks | Infra |
| 10.5 | 前端 SDK 完善（嵌入式集成包） | Frontend |
| 10.6 | 移动端接入（WebRTC 移动浏览器适配） | Frontend |
| 10.7 | 社交媒体渠道（微信/微博按需） | App |
| 10.8 | 安全审计（渗透测试 + 漏洞扫描） | Test |
| 10.9 | MySQL 分区实施（calls/call_events/agent_presence_log/im_messages/audit_logs/qa_results/webrtc_quality_logs/annotation_results 按月分区） | Infra |
| 10.10 | 监控告警（Prometheus + Grafana） | Infra |
| 10.11 | LLM 网关（多大模型接入: 百炼/第三方/自部署，统一 API 抽象） | Infra |
| 10.12 | 通信智能体（LLM Agent 自主接听来电/多轮对话/自主决策） | App |
| 10.13 | 声纹复刻/个性化音色（自定义 TTS 音色） | App |
| 10.14 | 智能对话分析（意图挖掘/销售话术提取/SOP发现 — 数据资产沉淀） | App |
| 10.15 | 智能培训（课程/考试/模拟通话练习） | App |
| 10.16 | 彩铃识别（外呼自动检测真人/语音信箱/忙线） | App |
| 10.17 | 全双工交互（智能打断/音色连续性） | App |
| 10.18 | **标注管理（标注中心/标注任务CRUD/标注工作台/数据集管理）** | App |
| 10.19 | **WebRTC 通话质量监控（SaveWebRtcInfo/Stats 采集 + 前端质量指示器）** | Infra |
| 10.20 | **视频通话扩展（media_type=VIDEO 预留, mod_verto 视频能力探索）** | Research |
| 10.21 | **前端: 数字员工 → 标注管理（标注中心/数据集/标注工作台）** | Frontend |
| 10.22 | **前端: 通话详情 → WebRTC 质量指标（丢包/抖动/延迟/MOS 红绿灯）** | Frontend |

### 退出标准
- [ ] 渗透测试通过
- [ ] SLO 99.9% 达标
- [ ] 1000 并发压测通过
- [ ] 多 FreeSWITCH 实例负载均衡
- [ ] TLS/SRTP 全链路加密
- [ ] MySQL 分区生效
- [ ] LLM 网关可切换多模型
- [ ] LLM Agent 可自主接听来电并对话
- [ ] 标注管理闭环可用（创建任务→标注→完成）
- [ ] WebRTC 通话质量可采集并展示（丢包/抖动/延迟/MOS）
- [ ] 视频通话 media_type 字段预留验证

---

## 全局功能清单交叉索引

确保 v5 方案中的每一项功能都有对应 Phase：

| v5 模块 | 功能 | Phase |
|---|---|---|
| A1 坐席管理 | CRUD/批量/SIP话机初始化 | 0, 5 |
| A2 技能组 | CRUD/成员管理/路由策略 | 0 |
| A3 号码管理 | 列表/绑定IVR/用途/分组 | 1 |
| A4 SIP中继 | CRUD/健康检查/Failover | 1, 6 |
| A5 中继路由 | 路由规则匹配引擎 | 2 |
| A6 IVR 流程 | 20种节点/画布/编辑锁/克隆/版本 | 1 |
| A7 坐席工作台 | 软电话条/Hold/Mute/Transfer/会议/监听/二次拨号/工作模式 | 1, 3, 5 |
| A8 预测式外呼 | 活动/案例/4种模式 | 6 |
| A9 双呼B2B | 背靠背通话 | 6 |
| A10 通话录音 | 录音/存储/播放/下载 | 1 |
| A13 语音信箱 | 列表/播放/标记已读/删除 | 1 |
| A11 呼入控制 | 号码标签/自动打标/黑名单/DNC | 1, 2 |
| A12 外呼多模式 | Preview/Progressive/Power | 6 |
| B1 多渠道接入 | 网页/APP/小程序/Email | 8 |
| B2 在线会话路由 | 技能组分配 | 8 |
| B3 坐席在线工作台 | 多会话并发 | 8 |
| B4 营业时间管理 | CRUD/节假日 | 0 |
| C1 客户管理 | 多号码/自定义字段/批量导入导出 | 7 |
| C2 工单管理 | 类目/模板/字段/流程节点/上线下线 | 7 |
| C3 会话信息 | 字段管理/模板管理 | 7 |
| C4 知识库 | 分类/文章/全文搜索 | 7 |
| C5 坐席脚本 | 话术引导 | 7 |
| D1 实时监控 | 7大指标/漏斗图/AI概览/大屏 | 4 |
| D2 坐席报表 | 30+字段/5维度 | 4 |
| D3 分组坐席报表 | 按技能组维度 | 4 |
| D4 技能组报表 | | 4 |
| D5 坐席状态日志 | 小休码筛选 | 4 |
| D6 热词/标签分析 | | 9 |
| D7 满意度调查 | IVR+SMS | 4 |
| D8 绩效管理 | 记分卡 | 9 |
| E1 数字员工 | 意图/FAQ/对话流/IVR集成 | 9 |
| E2 智能质检 | 录音转文字+规则 | 9 |
| E3 坐席辅助 | 实时ASR+推荐 | 9 |
| E4 热词/情绪/会话标签分析 | 实时+离线 | 9 |
| E5 AI摘要 | 点击生成+自动生成 | 9 |
| E6 AI满意度预测 | 无需客户参与 | 9 |
| E7 ASR热词库 | 租户自定义 | 9 |
| E8 快捷回复 | 全局/技能组/坐席 | 3 |
| E9 质检申诉/复核 | 申诉→复核闭环 | 9 |
| E10 智能对话分析 | 数据资产沉淀 | 10 |
| E11 LLM网关 | 多大模型接入 | 10 |
| E12 声纹复刻 | 个性化音色 | 10 |
| E13 智能培训 | 课程/考试/模拟 | 10 |
| E14 彩铃识别 | 真人/信箱/忙线检测 | 10 |
| F1 前端SDK | 嵌入式集成 | 10 |
| F2 事件推送 | Webhook/MQ | 3 |
| F3 SIP话机 | 注册直连 | 5 |
| F4 移动端 | WebRTC移动适配 | 10 |
| F5 短信/闪信 | 阿里云SMS/配置管理/按需发送 | 3, 6 |
| F6 加密通话/号码脱敏 | 隐私保护 + 号码脱敏外呼 | 6 |
| F7 来电弹屏 | iframe/5个/HTTPS | 3 |
| F8 WebRTC通话质量监控 | 丢包/抖动/延迟/MOS | 10 |
| G1 多租户 | 实例管理 | 0 |
| G2 权限/RBAC | JWT+角色 | 0 |
| G3 音频管理 | 上传/播放/分类 | 0 |
| G4 系统配置 | 租户设置 | 0 |
| G5 审计日志 | 所有写操作 | 0 |
| G6 API限流 | per-tenant | 0 |
| G7 录音存储配置 | 本地/MinIO/租户OSS | 0 |
| E15 标注管理 | 标注中心/数据集/标注工作台 | 10 |

### 数据表归属

| 表名 | Phase | BC |
|---|---|---|
| tenants | 0 | Identity |
| tenant_settings | 0 | Identity |
| users | 0 | Identity |
| agents | 0 | Identity |
| agent_presence_log | 0(schema), 3(DIALING+子状态增强) | Identity |
| skill_groups | 0 | Identity |
| skill_group_members | 0 | Identity |
| carriers | 1 | Telephony |
| sip_trunks | 1 | Telephony |
| sip_trunk_groups | 1 | Telephony |
| sip_trunk_group_members | 1 | Telephony |
| phone_numbers | 1 | Telephony |
| phone_number_skill_groups | 1 | Telephony |
| phone_number_dedicated_agents | 1 | Telephony |
| audio_files | 0 | Operation |
| ivr_flows | 1 | Routing |
| ivr_flow_versions | 1 | Routing |
| trunk_routing_rules | 2 | Telephony |
| cli_selection_policies | 2 | Telephony |
| break_reasons | 0 | Configuration |
| disposition_codes | 0 | Configuration |
| call_tags | 0 | Configuration |
| call_number_tags | 1 | Configuration |
| auto_tag_rules | 1 | Configuration |
| dnc_list | 2 | Integration |
| calls | 1(schema), 2(时长字段填充) | Call |
| call_events | 1 | Call |
| ivr_tracking | 1 | Call |
| call_tag_assignments | 2 | Call |
| queue_snapshots | 1 | Call |
| callback_requests | 3 | Call |
| campaigns | 6 | Campaign |
| campaign_cases | 6 | Campaign |
| customers | 7 | CRM |
| customer_phones | 7 | CRM |
| customer_interactions | 7 | CRM |
| ticket_categories | 7 | Ticket |
| ticket_templates | 7 | Ticket |
| tickets | 7 | Ticket |
| ticket_comments | 7 | Ticket |
| knowledge_categories | 7 | AI |
| knowledge_articles | 7 | AI |
| agent_scripts | 7 | AI |
| im_channels | 8 | IM |
| im_sessions | 8 | IM |
| im_messages | 8 | IM |
| business_hours | 0 | Operation |
| business_hours_schedule | 0 | Operation |
| digital_employees | 9 | AI |
| digital_employee_scenes | 9 | AI |
| recordings | 1 | Call |
| voicemails | 1 | Operation |
| csat_configs | 4 | Integration |
| csat_results | 4 | Integration |
| session_info_templates | 7 | Integration |
| custom_field_definitions | 0 | Configuration |
| screen_pop_configs | 3 | Integration |
| webhook_configs | 3 | Integration |
| webhook_delivery_log | 3 | Integration |
| audit_logs | 0 | Platform |
| performance_scorecards | 9 | Report |
| qa_rules | 9 | AI |
| qa_schemes | 9 | AI |
| qa_scheme_rules | 9 | AI |
| qa_results | 9 | AI |
| quick_replies | 3 | Configuration |
| asr_hotwords | 9 | AI |
| annotation_tasks | 10 | AI |
| annotation_results | 10 | AI |
| webrtc_quality_logs | 10 | Call |

**全部 69 张表已分配到 Phase 0-10，无遗漏。**

---

*文档版本: 1.4 | 对应方案: v5.2 | 全 11 Phase 规划完成 | 所有功能/表/API/前端页面已交叉索引 | v1.1: 修复遗漏 | v1.2: 拨号模式细化 + 高并发优化 + 可观测性前置 | v1.3: AI全面增强(+21功能+6表) | v1.4: 第四轮全功能对标(+13项增强+3表, 总计69表)*
