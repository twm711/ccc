# 第三轮 Review — 阿里云 CCC AI 功能全网深度探究 + v5 遗漏分析

---

## 一、阿里云 CCC AI 功能完整清单

基于阿里云官方文档、产品页、帮助中心的全面探究，阿里云联络中心 AI 体系分为以下几大产品线：

### 产品线 1：云联络中心 (CCC) — 语音业务 AI 功能

| # | 功能 | 说明 |
|---|---|---|
| 1 | **实时语音转写** | 通话中实时将语音转为文字，显示在坐席工作台 |
| 2 | **热词分析** | 按时间段/技能组 AI 提取对话热词，生成报表 |
| 3 | **会话标签分析** | 自定义标签 + AI 自动对会话进行语义分类标注 |
| 4 | **客户情绪检测** | 通话中实时检测客户情绪状态 |
| 5 | **满意度检测** | AI 预测满意度（非 IVR 评分，而是 AI 分析得出的） |
| 6 | **实时辅助/话术推荐** | 热线通话中基于销售流程实时推荐话术 |
| 7 | **质检推送** | 质检结果推送到坐席/管理员 |
| 8 | **批量预测式外呼 AI 预测** | AI 预测坐席可用时间，自动控制外呼节奏 |

### 产品线 2：云联络中心 (CCC) — 网络业务 AI 功能

| # | 功能 | 说明 |
|---|---|---|
| 9 | **会话信息自动生成** | AI 自动生成通话/会话小结（支持"坐席点击生成"和"后台自动生成"两种模式） |
| 10 | **自动填单** | AI 从对话中提取信息自动填充工单模板字段 |
| 11 | **IM 坐席辅助** | 在线客服消息 AI 纠错 / AI 扩写 / 话术优化 |
| 12 | **快捷回复** | 预置话术模板管理（按通用/技能组/坐席维度） |

### 产品线 3：智能对话分析 (SCA) — 质检体系

| # | 功能 | 说明 |
|---|---|---|
| 13 | **大模型质检规则** | 基于 LLM 的智能质检，理解复杂语境 + 长文本分析 |
| 14 | **普通质检规则（算子体系）** | 基于预定义算子的质检 |
| 15 | — 文字检查算子 | 关键词、文字相似度、正则、上下文重复、信息实体 |
| 16 | — 语音检查算子 | 静音检查、语速检查、抢话检查、角色判断、非正常挂机/接听、录音时长、能量检测、对话数量 |
| 17 | — 模型检查算子 | 客户检测模型、客服检测模型 |
| 18 | **实时质检** | 通话中实时转写 → 实时规则检查 → 事中干预 |
| 19 | **离线质检** | 通话后批量质检（语音/文本） |
| 20 | **二次质检** | 对已质检结果进行再次质检 |
| 21 | **复核/申诉** | 人工复核质检结果 + 坐席可对结果申诉 |
| 22 | **质检方案** | 多规则组合成质检方案，支持行业预置模板 |
| 23 | **智能对话分析** | 深度分析通话数据 → 意图提取、金牌话术沉淀、SOP 挖掘、FAQ 提炼 → 数据资产 |

### 产品线 4：智能联络中心 (AICCS) — 新一代 AI 产品

| # | 功能 | 说明 |
|---|---|---|
| 24 | **通信智能引擎** | LLM 网关配置（支持百炼/第三方/自有大模型）+ 大模型应用管理 + 通话事件感知（打断/静默/抢话/彩铃识别/转坐席） |
| 25 | **ASR 热词库** | 语音识别热词自定义，改善特定词汇识别效果 |
| 26 | **个性化音色/声纹复刻** | 采集语音样本 → LLM 生成个性化拟真语音 |
| 27 | **通信智能体** | 开箱即用 LLM 语音呼叫方案（私有知识库 + 呼叫任务管理） |
| 28 | **小记智能体** | 通话后自动分析（ASR+NLU → 关键信息提取 + 摘要标签 + 质检结果） |
| 29 | **智能联络机器人（小模型）** | 画布式话术编排 + 知识库 + 话术模板市场 |

### 产品线 5：其他 AI 辅助

| # | 功能 | 说明 |
|---|---|---|
| 30 | **智能培训** | 坐席培训/考试/模拟通话（AICCS 人工坐席功能中明确列出） |
| 31 | **彩铃识别** | 外呼时识别彩铃/忙音/留言/真人接听，优化预测外呼效果 |
| 32 | **数字员工全双工** | 全语音双工技术 — 智能打断、语气承接、拟人交互 |

---

## 二、v5 方案覆盖对照

| # | 阿里云功能 | v5 覆盖状态 | 差距 |
|---|---|---|---|
| 1 | 实时语音转写（工作台显示） | ❌ **缺失** | v5 有 ASR 节点但仅用于 IVR，无坐席通话实时转写 |
| 2 | 热词分析 | ✅ E4 + 9.7 | 已覆盖 |
| 3 | 会话标签分析（AI 语义标签） | ❌ **缺失** | v5 仅有号码级别 `call_tags`，无 AI 语义会话标签 |
| 4 | 客户情绪检测 | ✅ E4 + 9.9 | 已覆盖 |
| 5 | AI 满意度预测 | ❌ **缺失** | v5 仅有 IVR/SMS 满意度收集，无 AI 预测满意度 |
| 6 | 实时话术推荐（热线） | ❌ **缺失** | v5 `agent_scripts` 是静态脚本，非实时 AI 推荐 |
| 7 | 质检推送 | ⚠️ 部分 | 9.14 有前端任务，但无推送机制和配置 |
| 8 | AI 预测式外呼 | ✅ Phase 6 | 已覆盖 |
| 9 | 会话小结/AI 摘要 | ✅ E5 + 9.8 | 已覆盖，但缺少"坐席点击生成 vs 自动生成"两种模式 |
| 10 | 自动填单（AI→工单） | ❌ **缺失** | 无 |
| 11 | IM 坐席辅助（AI 纠错/扩写/优化） | ❌ **缺失** | Phase 8 在线客服无 AI 辅助 |
| 12 | 快捷回复模板 | ❌ **缺失** | 无快捷回复管理功能 |
| 13 | 大模型质检规则 | ❌ **缺失** | v5 仅有规则质检，无 LLM 质检 |
| 14-17 | 质检算子详细体系 | ⚠️ **不完整** | v5 仅写"录音转文字+规则分析"，未定义具体算子类型 |
| 18 | 实时质检 | ❌ **缺失** | v5 仅有离线质检 |
| 19 | 离线质检 | ✅ E2 + 9.5 | 已覆盖 |
| 20 | 二次质检 | ❌ **缺失** | 无 |
| 21 | 复核/申诉 | ❌ **缺失** | 无 |
| 22 | 质检方案 | ❌ **缺失** | 无规则组合成方案的概念 |
| 23 | 智能对话分析（数据资产） | ❌ **缺失** | 无意图挖掘/金牌话术/SOP 沉淀 |
| 24 | LLM 网关/大模型集成 | ❌ **缺失** | v5 仅有阿里通义作为 ASR/TTS，无 LLM 网关概念 |
| 25 | ASR 热词库 | ❌ **缺失** | 无热词库管理 |
| 26 | 声纹复刻/个性化音色 | ❌ **缺失** | 无 |
| 27 | 通信智能体（LLM Agent） | ❌ **缺失** | v5 数字员工更接近小模型方案 |
| 28 | 小记智能体 | ❌ **缺失** | 无 |
| 29 | 智能联络机器人（小模型） | ✅ E1 | v5 数字员工覆盖 |
| 30 | 智能培训 | ❌ **缺失** | 无培训/考试/模拟 |
| 31 | 彩铃识别 | ❌ **缺失** | 无 |
| 32 | 全双工交互 | ⚠️ 部分 | v5 有打断但未定义全双工/语气承接 |

**统计：32 项 AI 功能中，v5 完整覆盖 7 项、部分覆盖 4 项、完全缺失 21 项。**

---

## 三、遗漏分类与建议

### 🔴 A 类 — 核心 AI 功能缺失（强烈建议加入 Phase 9）

| # | 功能 | 建议 Phase | 理由 |
|---|---|---|---|
| 1 | **实时语音转写** | Phase 9 | 坐席通话中显示实时文字，行业标准功能，提升效率 |
| 2 | **实时质检** | Phase 9 | 通话中实时检测违规/风险，事中干预 > 事后补救 |
| 3 | **实时话术推荐** | Phase 9 | 基于通话上下文实时推荐话术/知识点，销售场景核心 |
| 4 | **自动填单** | Phase 9 | AI 从对话提取信息填充工单，减少坐席话后处理时间 |
| 5 | **会话标签分析** | Phase 9 | AI 语义标签分类，数据资产沉淀 |
| 6 | **大模型质检** | Phase 9 | LLM 理解复杂语境，比规则质检更准确 |
| 7 | **质检规则/算子详细定义** | Phase 9 | 需定义具体算子类型（关键词/正则/静音/语速/抢话等） |

### 🟡 B 类 — 重要辅助功能（建议 Phase 8-9）

| # | 功能 | 建议 Phase | 理由 |
|---|---|---|---|
| 8 | **IM 坐席辅助（AI 文本优化）** | Phase 8 | 随 IM 一起做，AI 纠错/扩写/优化 |
| 9 | **快捷回复模板** | Phase 3 | 简单但实用，随坐席工作台一起做 |
| 10 | **质检申诉/复核** | Phase 9 | 质检闭环管理 |
| 11 | **质检方案模板** | Phase 9 | 多规则组合 + 行业预置 |
| 12 | **ASR 热词库** | Phase 9 | 改善特定行业/企业词汇识别 |
| 13 | **AI 满意度预测** | Phase 9 | 补充 IVR 满意度，无需客户参与 |

### 🟢 C 类 — 高级 AI 功能（Phase 10+ 或独立模块）

| # | 功能 | 建议 Phase | 理由 |
|---|---|---|---|
| 14 | **LLM 网关** | Phase 10+ | 支持多种大模型接入（百炼/第三方/自有），架构扩展 |
| 15 | **通信智能体（LLM Agent）** | Phase 10+ | 开箱即用 LLM 语音呼叫方案 |
| 16 | **声纹复刻/个性化音色** | Phase 10+ | 高级 TTS 能力 |
| 17 | **小记智能体** | Phase 10+ | 通话后自动分析 Agent |
| 18 | **智能对话分析（数据资产）** | Phase 10+ | 意图挖掘/金牌话术/SOP 沉淀 |
| 19 | **智能培训** | Phase 10+ | 培训/考试/模拟通话，独立子系统 |
| 20 | **彩铃识别** | Phase 10+ | 外呼优化，提升预测外呼效率 |
| 21 | **全双工/语气承接** | Phase 10+ | 高级拟人交互 |

---

## 四、需要新增/修改的数据模型

### 新增表（6 张，Phase 9 需要）

```sql
-- 质检规则
CREATE TABLE qa_rules (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128) NOT NULL,
  rule_type       ENUM('keyword','regex','similarity','silence','speed','interruption',
                       'energy','recording_length','entity','role_detect',
                       'abnormal_hangup','llm') NOT NULL,
  severity        ENUM('info','warning','critical') NOT NULL DEFAULT 'warning',
  config          JSON NOT NULL COMMENT '算子配置: {keywords, threshold, duration_sec等}',
  score_impact    INT NOT NULL DEFAULT -5 COMMENT '命中时扣分值',
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_tenant (tenant_id)
);

-- 质检方案（多规则组合）
CREATE TABLE qa_schemes (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128) NOT NULL,
  base_score      INT NOT NULL DEFAULT 100 COMMENT '基础分',
  is_template     BOOLEAN NOT NULL DEFAULT FALSE,
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_tenant (tenant_id)
);

-- 质检方案-规则关联
CREATE TABLE qa_scheme_rules (
  scheme_id       BIGINT UNSIGNED NOT NULL,
  rule_id         BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (scheme_id, rule_id)
);

-- 质检结果
CREATE TABLE qa_results (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  call_id         BIGINT UNSIGNED NOT NULL,
  scheme_id       BIGINT UNSIGNED,
  task_type       ENUM('realtime','offline','re_inspect') NOT NULL,
  score           INT NOT NULL,
  hit_rules       JSON COMMENT '[{rule_id, rule_name, hit_detail, score_impact}]',
  reviewer_id     BIGINT UNSIGNED COMMENT '复核人',
  review_status   ENUM('pending','reviewed','appealed','appeal_resolved') DEFAULT 'pending',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_tenant_call (tenant_id, call_id),
  INDEX idx_tenant_time (tenant_id, created_at)
);

-- 快捷回复模板
CREATE TABLE quick_replies (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  shortcut        VARCHAR(64) NOT NULL,
  content         TEXT NOT NULL,
  scope           ENUM('global','skill_group','agent') NOT NULL DEFAULT 'global',
  scope_id        BIGINT UNSIGNED COMMENT '技能组ID或坐席ID（scope=global时为NULL）',
  sort_order      INT NOT NULL DEFAULT 0,
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_tenant_scope (tenant_id, scope, scope_id)
);

-- ASR 热词库
CREATE TABLE asr_hotwords (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128) NOT NULL,
  words           JSON NOT NULL COMMENT '["词1","词2",...]',
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_tenant (tenant_id)
);
```

### 修改现有表

```sql
-- calls 表增加实时转写和 AI 分析字段
ALTER TABLE calls ADD COLUMN realtime_transcript_url VARCHAR(512) COMMENT '实时转写文本存储路径';
ALTER TABLE calls ADD COLUMN ai_satisfaction_score   TINYINT UNSIGNED COMMENT 'AI 预测满意度 (1-5)';
ALTER TABLE calls ADD COLUMN ai_session_tags         JSON COMMENT 'AI 自动标签 [{tag, confidence}]';
ALTER TABLE calls ADD COLUMN ai_summary_auto         TEXT COMMENT '自动生成的通话摘要';
```

---

## 五、实施计划修改建议

### Phase 3 修改（加入快捷回复）

```
| 3.18 | Application: 快捷回复 CRUD（按全局/技能组/坐席维度） | App |
| 3.19 | 前端: 设置 → 快捷回复管理 + 坐席工作台快捷回复面板 | Frontend |
```

### Phase 8 修改（IM 加入 AI 辅助）

```
| 8.x | Application: IM 坐席辅助 — AI 纠错/扩写/话术优化（LLM API 调用） | App |
| 8.x | 前端: 在线工作台 — AI 辅助面板（输入框上方） | Frontend |
```

### Phase 9 修改（大幅扩展 AI 功能）

现有 9.1-9.15 保留，新增：

```
| 9.16 | Infrastructure: 实时语音转写服务（通话中 ASR 流式识别 → WebSocket 推送到坐席） | Infra |
| 9.17 | Application: 实时质检引擎（转写文本 → 规则/LLM 实时检测 → 告警推送） | App |
| 9.18 | Application: 实时话术推荐（转写文本 → 知识库/脚本匹配 → 推送到坐席） | App |
| 9.19 | Application: 自动填单（通话/对话文本 → LLM 提取 → 工单字段自动填充） | App |
| 9.20 | Application: 会话标签分析（LLM 对通话内容自动分类标注） | App |
| 9.21 | Application: 大模型质检（LLM 分析质检 + 规则质检并行） | App |
| 9.22 | Application: 质检方案管理（多规则组合 + 行业模板） | App |
| 9.23 | Application: 质检申诉/复核流程 | App |
| 9.24 | Application: AI 满意度预测（无需客户参与） | App |
| 9.25 | Application: ASR 热词库管理 | App |
| 9.26 | HTTP Handlers: 质检规则/方案/结果/申诉 + 快捷回复 + 热词库 API | Interface |
| 9.27 | 前端: 坐席工作台 — 实时转写面板 + 实时话术推荐 + 自动填单按钮 | Frontend |
| 9.28 | 前端: 质检管理 — 规则/方案/结果列表/申诉管理 | Frontend |
| 9.29 | 前端: ASR 热词库管理 | Frontend |
```

### Phase 9 退出标准补充

```
- [ ] 坐席通话中可看到实时语音转写
- [ ] 实时质检可检测违规并告警
- [ ] 实时话术推荐可推送到坐席工作台
- [ ] AI 自动填单可将对话信息填入工单
- [ ] 会话标签 AI 自动分类
- [ ] 大模型质检 + 规则质检并行运行
- [ ] 质检申诉/复核闭环
- [ ] ASR 热词库可配置
```

### Phase 10+ 新增（高级 AI）

```
| 10.x | Application: LLM 网关 — 多大模型接入（百炼/第三方/自有） | App |
| 10.x | Application: 通信智能体 — LLM Agent 语音呼叫（私有知识库+任务管理） | App |
| 10.x | Application: 声纹复刻/个性化音色 | App |
| 10.x | Application: 智能对话分析 — 意图/金牌话术/SOP 挖掘 | App |
| 10.x | Application: 智能培训 — 课程/考试/模拟通话 | App |
| 10.x | Application: 彩铃识别 — 外呼自动检测真人/留言/忙音 | App |
```

---

## 六、总结

| 类别 | 数量 | 影响 |
|---|---|---|
| 已覆盖 | 7 | — |
| 部分覆盖 | 4 | 需补充细节 |
| A 类缺失（Phase 9 必加） | 7 | 行业标准 AI 功能 |
| B 类缺失（Phase 8-9） | 6 | 重要辅助，建议加入 |
| C 类缺失（Phase 10+） | 8 | 高级/可选，按需规划 |

**核心发现：v5 的 AI 模块 (Phase 9) 严重低估了阿里云 CCC 的 AI 能力深度。** 主要差距在于：

1. **实时 vs 离线**：v5 仅有离线质检，阿里云已全面实时化（实时转写/实时质检/实时辅助）
2. **规则 vs LLM**：v5 仅有规则质检，阿里云已支持大模型质检
3. **被动 vs 主动**：v5 的坐席辅助是被动查知识库，阿里云是主动推送（实时话术推荐/自动填单）
4. **数据沉淀**：v5 缺少 AI 会话标签分析和智能对话分析这类数据资产沉淀能力

**数据模型影响：** 新增 6 张表 (qa_rules, qa_schemes, qa_scheme_rules, qa_results, quick_replies, asr_hotwords)，总表数 60 → 66 张。

确认后直接修复到 v5 + 实施计划中。
