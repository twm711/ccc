# 全阶段实施规划 Review 报告

> 对 `full-implementation-plan.md` 与 v5 方案 (`ccc-final-proposal-v5.md`) 的完整交叉比对

---

## 审核结论

**整体评估：规划基本完整，发现 8 个问题（3 个遗漏 + 3 个不一致 + 2 个建议）**

60 张表全部分配 ✓  
v5 全部功能模块 (A1-G6) 有 Phase 映射 ✓  
DDD 限界上下文划分合理 ✓  
TDD 范围覆盖所有核心域 ✓  
退出标准可验证 ✓

---

## 🔴 遗漏（需修复）

### 1. 语音信箱（Voicemail）— 前端页面 + API 路由全部缺失

**v5 依据：**
- Part 8.1 导航结构明确列出 `话务报表 → 语音信箱`
- Part 4.18 有 `voicemails` 表 DDL
- 实施计划中 `voicemails` 表分配到 Phase 1 ✓

**缺失：**
- 无任何 Phase 包含 voicemail 的 API 路由（如 `GET /api/v1/voicemails`, `PATCH /api/v1/voicemails/{id}/read`）
- 无任何 Phase 的前端页面列出 "语音信箱" 页面

**建议：** 补充到 **Phase 1**（与录音同期），添加：
- API：`GET /api/v1/voicemails`, `GET /api/v1/voicemails/{id}`, `PATCH /api/v1/voicemails/{id}/read`, `DELETE /api/v1/voicemails/{id}`
- 前端：`话务报表 → 语音信箱（列表/播放/标记已读/删除）`
- 退出标准补充：IVR 语音留言信箱节点可录制留言并可在前端播放

---

### 2. 短信配置页面 — 前端页面缺失

**v5 依据：**
- Part 8.7 设置页 Tab 列表明确列出 `短信配置 | 短信签名 / 模板 / 渠道设置`
- Part 8.1 导航结构 `设置 → 短信配置`

**缺失：**
- 实施计划中无任何 Phase 的前端页面包含 "短信配置"
- F5（短信/闪信）在交叉索引中映射到 Phase 3, 6，但两个 Phase 都没有短信管理相关 API 或前端页面

**建议：** 补充到 **Phase 3**（与事件推送同期），添加：
- API：`POST /api/v1/sms-configs`, `GET /api/v1/sms-configs`, `PUT /api/v1/sms-configs/{id}`, `POST /api/v1/sms/send`（按需发送）
- 前端：`设置 → 短信配置（签名/模板/渠道设置）`

---

### 3. 小休原因 / 结案代码 / 营业时间 — 前端管理页面遗漏

**v5 依据：**
- Phase 0 有 break_reasons / disposition_codes / business_hours 的完整 API 路由 ✓
- 但 Phase 0 前端页面列表中没有这些实体的管理页面

**当前 Phase 0 前端页面：**
```
- 平台管理控制台 → 概览
- 平台管理控制台 → 实例管理
- 实例内 → 客服管理 → 坐席列表/CRUD
- 实例内 → 客服管理 → 技能组列表/CRUD/成员管理
- 实例内 → 流程管理 → 音频管理
- 实例内 → 设置 → 呼入控制（号码标签管理基础）
```

**缺少：**
- 实例内 → 设置 → 小休原因管理（CRUD）
- 实例内 → 设置 → 结案代码管理（CRUD）
- 实例内 → 设置 → 营业时间管理（CRUD + 节假日配置）
- 实例内 → 设置 → 自定义字段管理（CRUD，scope=customer/session_info/ticket）

**建议：** 补充到 Phase 0 前端页面列表

---

## 🟡 不一致（需确认）

### 4. Phase 3 API 路由包含 `attended-transfer`，但 v5 将 热转/咨询转 安排在 Phase 5

**v5 Part 5：**
- Phase 3：`Hold/Mute/冷转(技能组/坐席/外部号码), 直接转接(blind_transfer)` — 仅冷转/盲转
- Phase 5：`热转(咨询转接)` — 咨询转/热转

**实施计划 Phase 3 API：**
```
POST /api/v1/calls/{id}/attended-transfer   ← 这是热转，应该在 Phase 5
```

**实施计划 Phase 5 API：**
```
POST /api/v1/calls/{id}/consult
POST /api/v1/calls/{id}/consult-transfer
POST /api/v1/calls/{id}/consult-cancel
```

**问题：** `attended-transfer` 和 `consult-transfer` 功能重叠。严格按 v5，Phase 3 只应有 `blind-transfer`。

**建议：** 将 `attended-transfer` 从 Phase 3 移到 Phase 5，或与 `consult-transfer` 合并为同一端点。

---

### 5. 营业时间管理 (B4) Phase 归属矛盾

**交叉索引：** `B4 营业时间管理 | CRUD/节假日 | 0`  
**v5 Part 8.1 导航：** `网络业务 → 设置 → 营业时间` — 网络业务的设置页是 Phase 8

**问题：** API 和数据层在 Phase 0，但 UI 自然位置在 Phase 8 的网络业务设置下。

**建议：** Phase 0 创建独立的营业时间管理页面（设置子页），Phase 8 在网络业务设置中集成引用同一组件。

---

### 6. v5 表统计数字错误（v5 文档本身的 bug）

**v5 Part 4 统计表：**
- `CDR/Runtime | 7` — 实际只列了 6 个表名（calls, call_events, ivr_tracking, call_tag_assignments, queue_snapshots, callback_requests）
- `Integration | 6` — 实际只列了 5 个表名（session_info_templates, custom_field_definitions, screen_pop_configs, webhook_configs, webhook_delivery_log）

**影响：** 不影响实施计划（实际 DDL 和表分配正确，共 60 张表），但 v5 文档的统计汇总行存在计数错误。

---

## 🟢 建议（可选优化）

### 7. Phase 1 IVR Engine 任务粒度偏粗

**当前：** 任务 1.12 `Go IVR Engine（JSON DAG 解释器 + ESL 调度）` 是单个任务

**实际工作量：** 需实现 20 种节点的解释逻辑，每种节点的 config 解析、ESL 命令映射、出口路由都不同。这是 Phase 1 中最复杂的单项任务。

**建议：** 执行 Phase 1 时拆分为子任务：
```
1.12a  IVR 引擎核心 — DAG 遍历器 + 变量作用域 + 节点分发
1.12b  基础节点 — start/end/play/set_variable/branch/hangup_reason
1.12c  交互节点 — collect_dtmf/voicemail/satisfaction_rating
1.12d  路由节点 — transfer_to_agent/transfer_to_external/blind_transfer/callback
1.12e  集成节点 — function/http_request/json_parser/sms
1.12f  高级节点 — sub_flow/digital_employee/asr(Phase 9 桩)
```

---

### 8. queue_snapshots 无清理策略

`queue_snapshots` 表按秒/分钟级别采样，数据量会持续增长，但规划中无定义保留策略或清理定时任务。

**建议：** 在 Phase 4（实时监控+报表）或 Phase 10（规模化+加固）中添加：
- queue_snapshots 保留 7 天，超期清理
- 或在 Phase 10 的 MySQL 分区中包含此表

---

## 审核通过项

| 检查项 | 结果 |
|---|---|
| 60 张表全部分配到 Phase | ✓ 无遗漏 |
| v5 全部功能模块 (A1-G6, 47项) 有 Phase 映射 | ✓ 完整 |
| 20 种 IVR 节点全部覆盖 | ✓ Phase 1 |
| 16 种通话类型全部覆盖 | ✓ 分布在 Phase 1-6 |
| DDD 14 个限界上下文划分清晰 | ✓ |
| 核心域 TDD (Identity/Call/Routing/Campaign/CRM) | ✓ 测试用例详细 |
| 坐席状态体系 (9主状态+5子状态+3工作模式+3系统小休码) | ✓ |
| 事件推送体系 (30+事件类型) | ✓ Phase 3 |
| 前端 Part 8 导航结构 vs Phase 分配 | ⚠ 3处遗漏 |
| Phase 退出标准可验证 | ✓ |
| 技术选型一致 (ESL/mod_verto/Keycloak/Snowflake/chi) | ✓ |
| 数据表归属 BC 一致 | ✓ |

---

## 结论

**需修复 3 个遗漏 + 确认 3 个不一致后，即可开始 Phase 0 实现。**

修复工作量：~15 分钟（补充 API 路由 + 前端页面列表 + 调整 attended-transfer Phase 归属）
