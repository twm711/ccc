# CCC 项目深度分析报告（第5轮）

**分析日期**: 2026-05-24
**分析基线**: main 分支 (commit c084064, PR #1 已合并)
**方法论**: 逐文件审查后端路由/service/handler + 前端API调用逐条比对 + 编译/测试/vet 全通过

---

## 一、综合成熟度评估

| 维度 | 评分 | 说明 |
|------|------|------|
| 数据层（MySQL Repos） | 92% | 82张表，50+ repo，SQL 真实完整 |
| 域逻辑（Domain Services） | 85% | Call 状态机 628 行，Campaign 4 种模式，Identity RBAC |
| API 路由覆盖 | 88% | 316 个路由端点（含 CRUD + 高级功能） |
| 基础设施（ESL/Redis/LLM） | 75% | ESL 已有 TCP 连接，Redis Dashboard 已有聚合查询 |
| **前后端对接** | **55%** | **约 25 处路由不匹配或缺失** |
| 端到端可运行 | **35%** | 无登录入口 → 所有 API 返回 401 |

**综合成熟度: ~72%**（后端扎实，但前后端集成断裂严重）

---

## 二、发现问题清单（按严重等级排序）

### 🔴 P0 — 系统无法启动使用（2个）

#### BUG-1: 缺少登录接口
- **现状**: 前端 `POST /api/v1/auth/login` (web/src/pages/Login.tsx:13)，后端无此路由
- **影响**: 系统无法登录，所有受保护 API 返回 401
- **需要**: AuthHandler + JWT 签发 + User.PasswordHash 字段 + 数据库迁移
- **相关文件**: router.go (缺公开路由), identity/entity.go (缺 PasswordHash), user_repo.go (缺 FindByUsernameGlobal)

#### BUG-2: 缺少 CORS 中间件
- **现状**: 前端通常运行在 localhost:5173，后端 localhost:8080，跨域请求被浏览器阻止
- **影响**: 即使登录接口存在，浏览器也会阻止 preflight 请求
- **需要**: middleware/cors.go + router.go 注册

### 🟡 P1 — 前后端路由不匹配导致 404（18个路由）

#### BUG-3: Dashboard 路由名称错误（4个）
| 前端调用 | 后端注册 | 状态 |
|----------|----------|------|
| `/dashboard/agents` | `/dashboard/agent-status` | 404 |
| `/dashboard/skill-groups` | `/dashboard/skill-group-status` | 404 |
| `/dashboard/trend` | `/dashboard/call-trend` | 404 |
| `/dashboard/funnel` | `/dashboard/call-funnel` | 404 |

#### BUG-4: Reports 路由名称错误（6个）
| 前端调用 | 后端注册 | 状态 |
|----------|----------|------|
| `/reports/agents` | `/reports/agent` | 404 |
| `/reports/agents/export` | `/reports/agent/export` | 404 |
| `/reports/skill-groups` | `/reports/skill-group` | 404 |
| `/reports/skill-groups/export` | `/reports/skill-group/export` | 404 |
| `/reports/internal-calls` | `/reports/internal-call` | 404 |
| `/reports/group-agents` | `/reports/group-agent` | 404 |

#### BUG-5: Knowledge 路由结构不匹配（2个）
| 前端调用 | 后端注册 | 状态 |
|----------|----------|------|
| `/knowledge/categories` | `/knowledge-categories` | 404 |
| `/knowledge/articles` | `/knowledge-articles` | 404 |

#### BUG-6: IM 路由结构不匹配（2个）
| 前端调用 | 后端注册 | 状态 |
|----------|----------|------|
| `/im/channels` | `/im-channels` | 404 |
| `/im/sessions` | `/im-sessions` | 404 |

#### BUG-7: Social Channel 路由路径不匹配
| 前端调用 | 后端注册 | 状态 |
|----------|----------|------|
| `/social-channels` (CRUD) | `/social-configs` | 404 |

#### BUG-8: AgentPresence 缺少 List 和 ChangeStatus 端点
| 前端调用 | 后端注册 | 状态 |
|----------|----------|------|
| `GET /agent-presence` (列表) | 不存在 | 404 |
| `POST /agent-presence/status` | 不存在 | 404 |

#### BUG-9: Voicemail 标记已读 HTTP 方法不匹配
| 前端调用 | 后端注册 | 状态 |
|----------|----------|------|
| `PUT /voicemails/{id}/read` | `PATCH /voicemails/{id}/read` | 405 |

### 🟡 P1 — 前端调用但后端完全缺失的端点（10个）

#### BUG-10: 缺少的后端端点
| 前端调用 | 文件位置 | 说明 |
|----------|----------|------|
| `GET /tenant-settings` | TenantSettingsPage.tsx:9 | 缺路由+handler |
| `PUT /tenant-settings` | TenantSettingsPage.tsx:14 | 缺路由+handler |
| `GET /supervisor/active-calls` | SupervisorPanel.tsx:27 | 缺路由+handler |
| `POST /ai/analysis/realtime` | AiAssistPanel.tsx:27 | 缺路由+handler |
| `GET /screen-pop/lookup` | ScreenPopPanel.tsx:25 | 缺路由+handler |
| `GET /campaigns/preview/current` | PreviewCaseCard.tsx:29 | 缺路由+handler |
| `POST /campaigns/{id}/cases/{caseId}/dial` | PreviewCaseCard.tsx:59 | 缺路由 |
| `POST /campaigns/{id}/cases/{caseId}/skip` | PreviewCaseCard.tsx:70 | 缺路由 |
| `POST /campaigns/{id}/resume` | endpoints.ts:91 | 缺路由（有 start/pause/abort，缺 resume） |
| `GET /me/profile` | MyWorkbenchPage.tsx:12 | 后端只有 PUT /me/profile，GET 404 |

#### BUG-11: Advanced AI 前后端路径完全不一致
| 前端调用 | 后端注册 | 说明 |
|----------|----------|------|
| `/ai/voice-clone/tasks` | `/voice-profiles` | 路径完全不同 |
| `/ai/conversation-analytics/analyze` | `/conversation-analysis` | 路径完全不同 |
| `/ai/training/generate-questions` | `/training/courses` | 路径+操作完全不同 |
| `/ai/training/evaluate` | `/training/exams` | 路径完全不同 |
| `GET /ai/script-recommend/{callId}` | `POST /calls/{callId}/script-recommendations` | 路径+方法均不同 |

#### BUG-12: Webchat 路径不匹配
| 前端调用 | 后端注册 | 说明 |
|----------|----------|------|
| `/webchat/sessions` | `/api/v1/widget/sessions` (公开) | baseURL 已包含 /api/v1，导致重复 |

#### BUG-13: CSAT 获取配置路径不匹配
| 前端调用 | 后端注册 | 说明 |
|----------|----------|------|
| `GET /csat/config` | `GET /csat/` | 路径不同 |

### 🟡 P2 — 后端逻辑/架构缺陷（5个）

#### BUG-14: WebSocket 路由未注册
- Dashboard Hub 和 IM Hub 已实现（gorilla/websocket）
- `cmd/server/main.go` 未创建 Hub 实例，也未启动 goroutine
- `router.go` 无 `/api/v1/ws/*` 路由
- 实时数据推送完全不工作

#### BUG-15: ASR/TTS Provider 创建后丢弃
- `main.go:231-232`: `_ = asrProvider` / `_ = ttsProvider`
- IVR ASR 节点只录音，不做语音识别
- IVR engine 的 `DefaultEngine()` 不接受 Transcriber 参数

#### BUG-16: 无优雅关机
- `main.go` 使用 `http.ListenAndServe()`，无 SIGINT/SIGTERM 处理
- 进程被 kill 时 WebSocket/DB 连接不会正常关闭

#### BUG-17: AttendedTransfer 无 ESL 调用
- `BlindTransfer` 调用了 `s.telephony.TransferCall()`（通过 adapter）
- `AttendedTransfer` 只更新数据库状态，不调 ESL

#### BUG-18: AgentPresenceService 缺 ListByTenant 方法
- 前端需要列出所有坐席状态（agentPresenceApi.list）
- service 只有 GetPresence(单个)，无批量查询

---

## 三、已修复（PR #1 解决的问题）

| 项目 | 状态 |
|------|------|
| ESL client 真实 TCP 连接 | ✓ 已有 net.DialTimeout + auth 握手 |
| ESL → CallService 接线 | ✓ TelephonyProvider adapter 已接入 |
| Advanced AI Providers 接线 | ✓ 6个 SetProvider() 已调用 |
| 5 个配置实体 CRUD | ✓ BreakReason/DispositionCode/AudioFile/BusinessHours/CallTag |
| QueueSnapshotRepo MySQL | ✓ |
| AuditLog handler + route | ✓ |
| SocialChannel ListConfigs/UpdateConfig | ✓ |
| IVR 20 节点真实 ESL 命令 | ✓ |
| Dashboard Redis 聚合 | ✓ |
| Aliyun ASR/TTS 真实实现 | ✓ |

---

## 四、修复工作量估算

| 优先级 | Bug 数量 | 预计工作量 |
|--------|----------|-----------|
| P0 (无法使用) | 2 | ~2 小时 |
| P1 (404/405) | 11 | ~4 小时 |
| P2 (架构缺陷) | 5 | ~3 小时 |
| **合计** | **18** | **~9 小时** |

---

## 五、代码质量亮点

1. **DDD 分层严格** — 所有业务逻辑在 domain 层，handler 只做 HTTP 转换
2. **测试覆盖** — 10 个 domain 包有单元测试，Mock repo 完整
3. **数据模型成熟** — 82 张表覆盖完整业务场景
4. **ESL 架构合理** — 连接池 + 熔断器 + 超时控制
5. **Provider 模式** — ASR/TTS/LLM/AI 全部可插拔

---

## 六、建议修复顺序

1. P0: 添加 /auth/login + CORS → 系统可登录
2. P1-路由: 修复 18 个路由名称/结构不匹配 → 前端页面可用
3. P1-缺失: 补充 10 个缺失端点 → 前端功能完整
4. P2: WebSocket 路由 + ASR 接线 + 优雅关机 → 核心功能可运行
