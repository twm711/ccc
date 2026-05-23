# Phase 0 实施规划

## 1. DDD 领域划分

### 1.1 限界上下文 (Bounded Contexts)

```
┌─────────────────────────────────────────────────────────────────┐
│                     CCC Platform                                 │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │  Identity BC  │  │  Routing BC  │  │  Telephony Asset BC  │   │
│  │  (核心域)      │  │  (核心域)    │  │  (支撑域)             │   │
│  │               │  │              │  │                       │   │
│  │  - Tenant     │  │  - IVR Flow  │  │  - Carrier            │   │
│  │  - User       │  │  - Routing   │  │  - SIP Trunk          │   │
│  │  - Agent      │  │    Rule      │  │  - Phone Number       │   │
│  │  - SkillGroup │  │  - CLI       │  │                       │   │
│  │               │  │    Policy    │  │                       │   │
│  └──────┬───────┘  └──────────────┘  └──────────────────────┘   │
│         │                                                         │
│  ┌──────┴───────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │  Platform BC  │  │ Operation BC │  │  Configuration BC    │   │
│  │  (支撑域)     │  │ (通用域)      │  │  (通用域)             │   │
│  │               │  │              │  │                       │   │
│  │  - Auth/JWT   │  │  - Audio     │  │  - Break Reason       │   │
│  │  - RBAC       │  │  - Business  │  │  - Disposition Code   │   │
│  │  - Rate Limit │  │    Hours     │  │  - Custom Field Def   │   │
│  │  - Audit Log  │  │              │  │                       │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 聚合 (Aggregates)

**Identity BC (核心域):**

| 聚合根 | 实体 | 值对象 |
|---|---|---|
| `Tenant` | TenantSettings | TenantCode, TenantStatus |
| `User` | Agent | Email, Phone, UserRole, WorkMode, UserStatus |
| `SkillGroup` | SkillGroupMember | RoutingPolicy, SkillLevel, GroupStatus |

**Routing BC (核心域):**

| 聚合根 | 实体 | 值对象 |
|---|---|---|
| `IVRFlow` | IVRFlowVersion | FlowGraph(JSON DAG), FlowStatus, FlowType, NodeConfig |
| `RoutingRule` | — | MatchCondition, Priority |
| `CLIPolicy` | — | Strategy(JSON) |

**Telephony Asset BC (支撑域):**

| 聚合根 | 实体 | 值对象 |
|---|---|---|
| `Carrier` | — | Region |
| `SIPTrunk` | — | TrunkHealth, Transport, Codecs |
| `SIPTrunkGroup` | SIPTrunkGroupMember | Priority, Weight |
| `PhoneNumber` | PhoneNumberSkillGroup, PhoneNumberDedicatedAgent | UsageFlags, Region |

**Configuration BC (通用域):**

| 聚合根 | 实体 | 值对象 |
|---|---|---|
| `BreakReason` | — | ReasonCode, IsSystem |
| `DispositionCode` | — | Category |
| `CustomFieldDefinition` | — | FieldType, FieldScope |

**Operation BC (通用域):**

| 聚合根 | 实体 | 值对象 |
|---|---|---|
| `AudioFile` | — | AudioFormat, AudioCategory |
| `BusinessHours` | BusinessHoursSchedule | TimeRange, DayOfWeek |

**Platform BC (支撑域):**

| 聚合根 | 实体 | 值对象 |
|---|---|---|
| `AuditLog` | — | Action, ResourceRef, IPAddress |

### 1.3 领域事件 (Domain Events)

| 事件 | 触发条件 | 消费者 |
|---|---|---|
| `TenantCreated` | 租户创建 | Keycloak同步, 初始化系统小休码 |
| `TenantSuspended` | 租户停用 | 坐席强制下线, 通话中断 |
| `UserCreated` | 用户创建 | Keycloak用户同步 |
| `UserDisabled` | 用户停用 | 坐席强制下线 |
| `AgentCreated` | 坐席配置创建 | — |
| `SkillGroupUpdated` | 技能组配置变更 | ACD 队列刷新 |
| `SkillGroupMemberChanged` | 成员变更 | ACD 队列刷新 |

### 1.4 跨上下文关系

```
Identity BC ──(ACL)──→ Platform BC (JWT 验证时读取 tenant_id)
Identity BC ──(ACL)──→ Routing BC (IVR flow 引用 skill_group_id)
Identity BC ──(ACL)──→ Telephony BC (号码绑定坐席/技能组)
Platform BC ──(OHS)──→ 所有 BC (审计日志中间件拦截所有写操作)
```

---

## 2. 分层架构

```
Go 项目遵循 Clean Architecture / Hexagonal Architecture:

cmd/server/main.go           ← 应用入口
│
├── internal/
│   ├── domain/              ← 领域层（纯业务逻辑，零外部依赖）
│   │   ├── identity/        ← Identity BC
│   │   │   ├── entity.go        聚合根/实体/值对象定义
│   │   │   ├── repository.go    Repository 接口（Port）
│   │   │   ├── service.go       领域服务
│   │   │   └── service_test.go  TDD 测试
│   │   ├── routing/         ← Routing BC
│   │   ├── telephony/       ← Telephony Asset BC
│   │   ├── configuration/   ← Configuration BC
│   │   ├── operation/       ← Operation BC
│   │   └── platform/        ← Platform BC
│   │
│   ├── application/         ← 应用层（用例编排，事务管理）
│   │   ├── tenant_app.go
│   │   ├── user_app.go
│   │   ├── skillgroup_app.go
│   │   └── ...
│   │
│   ├── infrastructure/      ← 基础设施层（外部适配器实现）
│   │   ├── persistence/     ← MySQL Repository 实现
│   │   │   ├── mysql.go         DB 连接
│   │   │   ├── tenant_repo.go
│   │   │   ├── user_repo.go
│   │   │   └── ...
│   │   ├── auth/            ← Keycloak JWT 验证
│   │   ├── ratelimit/       ← Redis 令牌桶限流
│   │   └── snowflake/       ← Snowflake ID 生成
│   │
│   └── interfaces/          ← 接口层（HTTP Handler + 路由）
│       ├── http/
│       │   ├── router.go
│       │   ├── middleware/
│       │   │   ├── auth.go
│       │   │   ├── ratelimit.go
│       │   │   ├── audit.go
│       │   │   └── tenant.go
│       │   ├── tenant_handler.go
│       │   ├── user_handler.go
│       │   └── ...
│       └── dto/             ← 请求/响应 DTO
│           ├── tenant_dto.go
│           └── ...
│
├── migrations/              ← SQL schema migrations
│   ├── 000001_init_schema.up.sql
│   └── 000001_init_schema.down.sql
│
├── pkg/                     ← 可复用公共包
│   └── httputil/            ← HTTP 工具函数
│
└── docs/                    ← 文档
    └── phase0-plan.md
```

---

## 3. TDD 策略

### 3.1 核心域必须 TDD（先写测试）

| 核心域 | 测试覆盖 | TDD 方式 |
|---|---|---|
| Tenant 聚合 | 创建验证、状态转换、Settings 默认值 | 单元测试（纯领域，mock repo） |
| User/Agent 聚合 | 创建验证、角色权限、坐席配置 | 单元测试 |
| SkillGroup 聚合 | 成员管理、路由策略验证、等级约束 | 单元测试 |

### 3.2 支撑域使用集成测试

| 支撑域 | 测试方式 |
|---|---|
| JWT/RBAC | 中间件单元测试（mock JWT） |
| API 限流 | 单元测试（mock Redis） |
| 审计日志 | 中间件单元测试 |

### 3.3 通用域使用标准测试

| 通用域 | 测试方式 |
|---|---|
| 音频/营业时间/小休原因/结案代码/自定义字段 | Repository 接口 + Handler 测试 |

### 3.4 测试命名约定

```go
func TestTenantService_Create_Success(t *testing.T) { ... }
func TestTenantService_Create_DuplicateCode(t *testing.T) { ... }
func TestTenantService_Suspend_ActiveTenant(t *testing.T) { ... }
func TestSkillGroupService_AddMember_LevelOutOfRange(t *testing.T) { ... }
```

---

## 4. 技术选型（Phase 0 依赖）

| 用途 | 库 | 说明 |
|---|---|---|
| HTTP Router | `github.com/go-chi/chi/v5` | 轻量，符合 net/http 标准 |
| DB Driver | `github.com/go-sql-driver/mysql` | MySQL 标准驱动 |
| DB Access | `github.com/jmoiron/sqlx` | 结构体映射 |
| Migration | `github.com/golang-migrate/migrate/v4` | SQL 文件迁移 |
| Snowflake | `github.com/bwmarrin/snowflake` | 64位有序ID |
| Redis | `github.com/redis/go-redis/v9` | Redis 客户端 |
| JWT | `github.com/golang-jwt/jwt/v5` | JWT 解析验证 |
| JWKS | `github.com/MicahParks/keyfunc/v3` | Keycloak JWKS 动态获取 |
| Logging | `log/slog` (标准库) | 结构化日志 |
| Validation | `github.com/go-playground/validator/v10` | 请求验证 |
| Testing | `testing` + `github.com/stretchr/testify` | 断言/mock |
| Config | 环境变量（标准库） | 12-factor |

---

## 5. Phase 0 任务分解（执行顺序）

### Step 1: 项目骨架 (约30min)
- [ ] `go mod init`
- [ ] 目录结构创建
- [ ] 依赖安装
- [ ] `cmd/server/main.go` 骨架
- [ ] 配置加载

### Step 2: 数据库 Migration (约20min)
- [ ] `migrations/000001_init_schema.up.sql` — 全部60张表DDL
- [ ] `migrations/000001_init_schema.down.sql` — DROP 全部表
- [ ] migration 执行工具集成

### Step 3: 核心域 TDD — Identity BC (约60min)
- [ ] `domain/identity/entity.go` — Tenant, User, Agent, SkillGroup 聚合根
- [ ] `domain/identity/repository.go` — Repository 接口
- [ ] `domain/identity/service_test.go` — **先写测试**
  - Tenant: Create/Get/Update/Suspend/List + Settings
  - User: Create/Get/Update/Disable/List
  - Agent: Create/Update
  - SkillGroup: Create/Update/AddMember/RemoveMember/List
- [ ] `domain/identity/service.go` — 实现领域服务使测试通过

### Step 4: 核心域 TDD — Routing BC (约30min)
- [ ] `domain/routing/entity.go` — IVRFlow 聚合根
- [ ] `domain/routing/service_test.go` — 先写测试（流程状态机、版本管理、编辑锁）
- [ ] `domain/routing/service.go` — 实现

### Step 5: 基础设施层 (约40min)
- [ ] `infrastructure/persistence/mysql.go` — DB 连接
- [ ] `infrastructure/persistence/tenant_repo.go` — Tenant Repository 实现
- [ ] `infrastructure/persistence/user_repo.go` — User Repository 实现
- [ ] `infrastructure/persistence/skillgroup_repo.go` — SkillGroup Repository 实现
- [ ] `infrastructure/snowflake/snowflake.go` — Snowflake ID 生成

### Step 6: 支撑域 — Platform BC (约30min)
- [ ] `infrastructure/auth/jwt.go` — JWT 解析 + Keycloak JWKS
- [ ] `interfaces/http/middleware/auth.go` — Auth 中间件（含 tenant_id 提取）
- [ ] `interfaces/http/middleware/ratelimit.go` — Redis 令牌桶限流
- [ ] `interfaces/http/middleware/audit.go` — 审计日志记录
- [ ] 中间件单元测试

### Step 7: 应用层 (约20min)
- [ ] `application/tenant_app.go` — 用例编排（创建租户 → 初始化Settings → 审计日志）
- [ ] `application/user_app.go`
- [ ] `application/skillgroup_app.go`

### Step 8: 接口层 — HTTP Handler (约40min)
- [ ] DTO 定义
- [ ] `interfaces/http/router.go` — 路由注册
- [ ] `interfaces/http/tenant_handler.go`
- [ ] `interfaces/http/user_handler.go`
- [ ] `interfaces/http/skillgroup_handler.go`
- [ ] 通用域 Handler: 音频/营业时间/小休原因/结案代码/自定义字段

### Step 9: 集成 + PR (约10min)
- [ ] `cmd/server/main.go` 完成（依赖注入/启动）
- [ ] `go vet` / `go build` 通过
- [ ] 创建 PR

---

## 6. API 路由设计 (Phase 0 范围)

```
# 租户管理 (admin only)
POST   /api/v1/tenants
GET    /api/v1/tenants
GET    /api/v1/tenants/{id}
PUT    /api/v1/tenants/{id}
GET    /api/v1/tenants/{id}/settings
PUT    /api/v1/tenants/{id}/settings

# 用户管理 (tenant scoped)
POST   /api/v1/users
GET    /api/v1/users
GET    /api/v1/users/{id}
PUT    /api/v1/users/{id}
DELETE /api/v1/users/{id}

# 坐席配置 (tenant scoped)
POST   /api/v1/users/{id}/agent
GET    /api/v1/users/{id}/agent
PUT    /api/v1/users/{id}/agent

# 技能组 (tenant scoped)
POST   /api/v1/skill-groups
GET    /api/v1/skill-groups
GET    /api/v1/skill-groups/{id}
PUT    /api/v1/skill-groups/{id}
DELETE /api/v1/skill-groups/{id}
POST   /api/v1/skill-groups/{id}/members
DELETE /api/v1/skill-groups/{id}/members/{userId}
GET    /api/v1/skill-groups/{id}/members

# 音频管理 (tenant scoped)
POST   /api/v1/audio-files
GET    /api/v1/audio-files
GET    /api/v1/audio-files/{id}
DELETE /api/v1/audio-files/{id}

# 营业时间 (tenant scoped)
POST   /api/v1/business-hours
GET    /api/v1/business-hours
GET    /api/v1/business-hours/{id}
PUT    /api/v1/business-hours/{id}
DELETE /api/v1/business-hours/{id}

# 小休原因 (tenant scoped)
POST   /api/v1/break-reasons
GET    /api/v1/break-reasons
PUT    /api/v1/break-reasons/{id}
DELETE /api/v1/break-reasons/{id}

# 结案代码 (tenant scoped)
POST   /api/v1/disposition-codes
GET    /api/v1/disposition-codes
PUT    /api/v1/disposition-codes/{id}
DELETE /api/v1/disposition-codes/{id}

# 自定义字段定义 (tenant scoped)
POST   /api/v1/custom-fields
GET    /api/v1/custom-fields
PUT    /api/v1/custom-fields/{id}
DELETE /api/v1/custom-fields/{id}

# 审计日志 (tenant scoped, read only)
GET    /api/v1/audit-logs
```

---

## 7. Phase 0 退出标准

| 验收项 | 标准 |
|---|---|
| 核心域测试 | Tenant/User/Agent/SkillGroup 领域服务全部测试通过 |
| API 可用 | 所有上述路由可通过 curl 调用 |
| 认证 | JWT Bearer Token 验证生效，无 token 返回 401 |
| 多租户隔离 | 不同 tenant_id 数据严格隔离 |
| RBAC | admin 可管理所有，agent 仅可访问自己数据 |
| 审计日志 | 所有写操作自动记录审计日志 |
| API 限流 | per-tenant 限流生效，超限返回 429 |
| 构建 | `go build` + `go vet` + `go test ./...` 全部通过 |
