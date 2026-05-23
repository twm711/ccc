-- Auto-generated from v5.2 specification
-- 69 tables for CCC platform
-- Engine: InnoDB, Charset: utf8mb4

CREATE TABLE tenants (
  id              BIGINT UNSIGNED PRIMARY KEY,
  code            VARCHAR(64) NOT NULL UNIQUE,
  display_name    VARCHAR(128) NOT NULL,
  domain          VARCHAR(255),
  status          ENUM('active','suspended','deleted') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE tenant_settings (
  tenant_id              BIGINT UNSIGNED PRIMARY KEY,
  default_acw_seconds    INT UNSIGNED NOT NULL DEFAULT 30,
  ring_timeout_seconds   INT UNSIGNED NOT NULL DEFAULT 30,
  auto_break_on_no_answer BOOLEAN NOT NULL DEFAULT TRUE,
  hangup_policy          ENUM('agent_only','both','customer_only') NOT NULL DEFAULT 'both',
  recording_enabled      BOOLEAN NOT NULL DEFAULT TRUE,
  recording_announce     BOOLEAN NOT NULL DEFAULT FALSE,
  recording_retention_days INT UNSIGNED NOT NULL DEFAULT 365,
  csat_enabled           BOOLEAN NOT NULL DEFAULT FALSE,
  max_queue_size         INT UNSIGNED NOT NULL DEFAULT 200,
  familiar_agent_days    INT UNSIGNED NOT NULL DEFAULT 30,
  timezone               VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
  locale                 VARCHAR(16) NOT NULL DEFAULT 'zh-CN',
  api_rate_limit_per_sec INT UNSIGNED NOT NULL DEFAULT 100,
  max_concurrent_calls    INT UNSIGNED NOT NULL DEFAULT 100 COMMENT '租户最大并发通话数',
  CONSTRAINT fk_ts_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE users (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  user_name       VARCHAR(64) NOT NULL,
  display_name    VARCHAR(128) NOT NULL,
  employee_id     VARCHAR(64),
  email           VARCHAR(255) NOT NULL,
  phone           VARCHAR(32),
  landline        VARCHAR(32),
  role            ENUM('agent','skill_group_leader','admin','tenant_admin') NOT NULL DEFAULT 'agent',
  work_mode       ENUM('on_site','off_site','sip_phone') NOT NULL DEFAULT 'on_site',
  status          ENUM('active','disabled','deleted') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at      TIMESTAMP NULL,
  UNIQUE KEY uniq_tenant_username (tenant_id, user_name),
  INDEX idx_tenant_email (tenant_id, email),
  CONSTRAINT fk_users_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE agents (
  user_id         BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  extension       VARCHAR(16),
  sip_extension   VARCHAR(16),
  sip_password_enc VARBINARY(255),
  sip_device_status ENUM('none','offline','online') NOT NULL DEFAULT 'none',
  max_concurrent  TINYINT UNSIGNED NOT NULL DEFAULT 1,
  max_chat_slots  TINYINT UNSIGNED NOT NULL DEFAULT 5,
  acw_seconds     INT UNSIGNED NOT NULL DEFAULT 30,
  outbound_only   BOOLEAN NOT NULL DEFAULT FALSE,
  personal_outbound_number_id BIGINT UNSIGNED NULL COMMENT '坐席个人外呼号码（独立于技能组绑定）',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_agents_user FOREIGN KEY (user_id) REFERENCES users(id),
  INDEX idx_agent_tenant (tenant_id)
);

-- v5 增强: 新增 dialing 状态 + sub_state + work_mode
CREATE TABLE agent_presence_log (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  user_id         BIGINT UNSIGNED NOT NULL,
  state           ENUM('offline','online','idle','ringing','dialing','talking','acw','break','invisible') NOT NULL,
  sub_state       VARCHAR(32) NULL COMMENT 'Monitored/Consulted/Consulting/Conference/Monitoring',
  reason_code     VARCHAR(64),
  work_mode       ENUM('on_site','off_site','office_phone') NULL,
  entered_at      TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  left_at         TIMESTAMP(3) NULL,
  duration_ms     BIGINT GENERATED ALWAYS AS (TIMESTAMPDIFF(MICROSECOND, entered_at, left_at) / 1000) VIRTUAL,
  INDEX idx_user_time (user_id, entered_at),
  INDEX idx_tenant_state_time (tenant_id, state, entered_at)
);

CREATE TABLE skill_groups (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  code            VARCHAR(64) NOT NULL,
  name            VARCHAR(128) NOT NULL,
  description     VARCHAR(512),
  routing_policy  ENUM('longest_idle','least_utilized','round_robin','skill_level','familiar_agent') NOT NULL DEFAULT 'longest_idle',
  max_queue_size  INT UNSIGNED NOT NULL DEFAULT 100,
  max_wait_sec    INT UNSIGNED NOT NULL DEFAULT 300,
  overflow_target ENUM('voicemail','transfer','callback','reject') NOT NULL DEFAULT 'voicemail',
  overflow_transfer_target VARCHAR(128),
  queue_music_id  BIGINT UNSIGNED,
  ewt_announce_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ewt_announce_interval_sec INT UNSIGNED NOT NULL DEFAULT 60,
  callback_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  callback_threshold_sec INT UNSIGNED NOT NULL DEFAULT 120,
  whisper_enabled  BOOLEAN NOT NULL DEFAULT FALSE,
  whisper_audio_id BIGINT UNSIGNED,
  status          ENUM('active','disabled','deleted') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_tenant_code (tenant_id, code),
  CONSTRAINT fk_sg_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE skill_group_members (
  skill_group_id  BIGINT UNSIGNED NOT NULL,
  user_id         BIGINT UNSIGNED NOT NULL,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  level           TINYINT UNSIGNED NOT NULL DEFAULT 5,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (skill_group_id, user_id),
  INDEX idx_user (user_id),
  INDEX idx_tenant (tenant_id),
  CONSTRAINT fk_sgm_sg FOREIGN KEY (skill_group_id) REFERENCES skill_groups(id) ON DELETE CASCADE,
  CONSTRAINT fk_sgm_user FOREIGN KEY (user_id) REFERENCES users(id),
  CHECK (level BETWEEN 1 AND 10)
);

CREATE TABLE carriers (
  id              BIGINT UNSIGNED PRIMARY KEY,
  code            VARCHAR(32) NOT NULL UNIQUE,
  display_name    VARCHAR(128) NOT NULL,
  region          VARCHAR(32),
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sip_trunks (
  id                  BIGINT UNSIGNED PRIMARY KEY,
  tenant_id           BIGINT UNSIGNED NOT NULL,
  carrier_id          BIGINT UNSIGNED NOT NULL,
  code                VARCHAR(64) NOT NULL,
  direction           ENUM('inbound','outbound','both') NOT NULL DEFAULT 'both',
  far_end_host        VARCHAR(255) NOT NULL,
  far_end_port        SMALLINT UNSIGNED NOT NULL DEFAULT 5060,
  transport           ENUM('udp','tcp','tls') NOT NULL DEFAULT 'udp',
  sp_marker           VARCHAR(64),
  auth_username       VARCHAR(128),
  auth_password_enc   VARBINARY(255),
  realm               VARCHAR(255),
  local_bind_ip       VARCHAR(64),
  options_keepalive_s INT UNSIGNED NOT NULL DEFAULT 30,
  max_concurrent      INT UNSIGNED NOT NULL DEFAULT 100,
  codecs              JSON NOT NULL,
  status              ENUM('active','disabled','deleted') NOT NULL DEFAULT 'active',
  health              ENUM('healthy','degraded','down','unknown') NOT NULL DEFAULT 'unknown',
  last_options_ok_at  TIMESTAMP NULL,
  created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_tenant_code (tenant_id, code),
  INDEX idx_carrier (carrier_id),
  CONSTRAINT fk_trunk_carrier FOREIGN KEY (carrier_id) REFERENCES carriers(id),
  CONSTRAINT fk_trunk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE sip_trunk_groups (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  code            VARCHAR(64) NOT NULL,
  description     VARCHAR(255),
  UNIQUE KEY uniq_tenant_code (tenant_id, code)
);

CREATE TABLE sip_trunk_group_members (
  group_id        BIGINT UNSIGNED NOT NULL,
  trunk_id        BIGINT UNSIGNED NOT NULL,
  priority        SMALLINT NOT NULL DEFAULT 100,
  weight          SMALLINT NOT NULL DEFAULT 100,
  PRIMARY KEY (group_id, trunk_id),
  CONSTRAINT fk_stgm_group FOREIGN KEY (group_id) REFERENCES sip_trunk_groups(id) ON DELETE CASCADE,
  CONSTRAINT fk_stgm_trunk FOREIGN KEY (trunk_id) REFERENCES sip_trunks(id) ON DELETE CASCADE
);

CREATE TABLE phone_numbers (
  id                  BIGINT UNSIGNED PRIMARY KEY,
  tenant_id           BIGINT UNSIGNED NOT NULL,
  number              VARCHAR(32) NOT NULL,
  display_label       VARCHAR(128),
  region_country      CHAR(2) NOT NULL DEFAULT 'CN',
  region_province     VARCHAR(32),
  region_city         VARCHAR(64),
  carrier_id          BIGINT UNSIGNED,
  usage_flags         SET('inbound','outbound') NOT NULL DEFAULT 'inbound,outbound',
  inbound_ivr_id      BIGINT UNSIGNED,
  digital_employee_id BIGINT UNSIGNED,
  group_label         VARCHAR(64),
  status              ENUM('active','disabled','deleted') NOT NULL DEFAULT 'active',
  created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_tenant_number (tenant_id, number),
  INDEX idx_carrier (carrier_id),
  CONSTRAINT fk_pn_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_pn_carrier FOREIGN KEY (carrier_id) REFERENCES carriers(id),
  CONSTRAINT fk_pn_ivr FOREIGN KEY (inbound_ivr_id) REFERENCES ivr_flows(id)
);

CREATE TABLE phone_number_skill_groups (
  phone_number_id BIGINT UNSIGNED NOT NULL,
  skill_group_id  BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (phone_number_id, skill_group_id),
  CONSTRAINT fk_pnsg_pn FOREIGN KEY (phone_number_id) REFERENCES phone_numbers(id) ON DELETE CASCADE,
  CONSTRAINT fk_pnsg_sg FOREIGN KEY (skill_group_id) REFERENCES skill_groups(id) ON DELETE CASCADE
);

CREATE TABLE phone_number_dedicated_agents (
  phone_number_id BIGINT UNSIGNED NOT NULL,
  user_id         BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (phone_number_id, user_id),
  CONSTRAINT fk_pnda_pn FOREIGN KEY (phone_number_id) REFERENCES phone_numbers(id) ON DELETE CASCADE,
  CONSTRAINT fk_pnda_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE audio_files (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128) NOT NULL,
  file_path       VARCHAR(512) NOT NULL,
  duration_ms     INT UNSIGNED,
  format          ENUM('wav','mp3','ogg') NOT NULL DEFAULT 'wav',
  category        ENUM('ivr','queue_music','hold_music','whisper','system') NOT NULL DEFAULT 'ivr',
  status          ENUM('active','disabled','deleted') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_af_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- v5 增强: ivr_flows 新增更细粒度状态 + 编辑锁 + 克隆/导入/导出/版本管理
CREATE TABLE ivr_flows (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  code            VARCHAR(64) NOT NULL,
  name            VARCHAR(128) NOT NULL,
  flow_type       ENUM('main','sub','survey') NOT NULL DEFAULT 'main',
  version         INT UNSIGNED NOT NULL DEFAULT 1,
  graph           JSON NOT NULL,
  status          ENUM('draft','publishing','published','editing','published_with_draft','failed','archived') NOT NULL DEFAULT 'draft',
  locked_by       BIGINT UNSIGNED NULL COMMENT '编辑锁持有者 user_id',
  locked_at       TIMESTAMP NULL COMMENT '编辑锁获取时间',
  published_at    TIMESTAMP NULL,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_tenant_code_version (tenant_id, code, version)
);

-- v5 新增: IVR 版本历史表（支持回滚）
CREATE TABLE ivr_flow_versions (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  ivr_flow_id     BIGINT UNSIGNED NOT NULL,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  version         INT UNSIGNED NOT NULL,
  graph           JSON NOT NULL,
  description     VARCHAR(512),
  published_by    BIGINT UNSIGNED,
  published_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_flow_version (ivr_flow_id, version),
  CONSTRAINT fk_ifv_flow FOREIGN KEY (ivr_flow_id) REFERENCES ivr_flows(id)
);

CREATE TABLE trunk_routing_rules (
  id                 BIGINT UNSIGNED PRIMARY KEY,
  tenant_id          BIGINT UNSIGNED NOT NULL,
  name               VARCHAR(128) NOT NULL,
  priority           INT NOT NULL DEFAULT 100,
  match_cli_prefix          VARCHAR(32),
  match_cli_carrier_id      BIGINT UNSIGNED,
  match_cli_country         CHAR(2),
  match_dest_prefix         VARCHAR(32),
  match_dest_country        CHAR(2),
  match_dest_region         VARCHAR(64),
  match_time_of_day_start   TIME,
  match_time_of_day_end     TIME,
  match_dow_mask            TINYINT UNSIGNED,
  match_skill_group_id      BIGINT UNSIGNED,
  match_tenant_cost_class   ENUM('premium','standard','grey'),
  target_trunk_group_id     BIGINT UNSIGNED NOT NULL,
  cli_rewrite_rule          VARCHAR(255),
  enabled                   BOOLEAN NOT NULL DEFAULT TRUE,
  created_at                TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at                TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_tenant_priority (tenant_id, priority),
  INDEX idx_match_cli_prefix (tenant_id, match_cli_prefix),
  INDEX idx_match_dest_prefix (tenant_id, match_dest_prefix),
  CONSTRAINT fk_trr_target FOREIGN KEY (target_trunk_group_id) REFERENCES sip_trunk_groups(id),
  CONSTRAINT fk_trr_cli_carrier FOREIGN KEY (match_cli_carrier_id) REFERENCES carriers(id),
  CONSTRAINT fk_trr_skill_group FOREIGN KEY (match_skill_group_id) REFERENCES skill_groups(id)
);

CREATE TABLE cli_selection_policies (
  id                       BIGINT UNSIGNED PRIMARY KEY,
  tenant_id                BIGINT UNSIGNED NOT NULL,
  name                     VARCHAR(128) NOT NULL,
  strategy                 JSON NOT NULL,
  enabled                  BOOLEAN NOT NULL DEFAULT TRUE,
  created_at               TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_tenant_name (tenant_id, name)
);

CREATE TABLE break_reasons (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  code            VARCHAR(64) NOT NULL,
  name            VARCHAR(128) NOT NULL,
  is_system       BOOLEAN NOT NULL DEFAULT FALSE,
  sort_order      INT NOT NULL DEFAULT 0,
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_tenant_code (tenant_id, code),
  CONSTRAINT fk_br_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- v5: Phase 0 schema migration 初始化系统小休码
-- INSERT INTO break_reasons (tenant_id, code, name, is_system, sort_order) VALUES
--   (0, 'Warm-up', '上线预热', TRUE, -3),
--   (0, 'RingingTimeout', '振铃超时', TRUE, -2),
--   (0, 'RejectCall', '拒接小休', TRUE, -1);

CREATE TABLE disposition_codes (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  code            VARCHAR(64) NOT NULL,
  name            VARCHAR(128) NOT NULL,
  category        VARCHAR(64),
  sort_order      INT NOT NULL DEFAULT 0,
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_tenant_code (tenant_id, code),
  CONSTRAINT fk_dc_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE call_tags (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(64) NOT NULL,
  color           CHAR(7) DEFAULT '#3B82F6',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_tenant_name (tenant_id, name),
  CONSTRAINT fk_ct_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE call_number_tags (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  number          VARCHAR(32) NOT NULL,
  tag_id          BIGINT UNSIGNED NOT NULL,
  remark          VARCHAR(255),
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_cnt_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_cnt_tag FOREIGN KEY (tag_id) REFERENCES call_tags(id) ON DELETE CASCADE,
  INDEX idx_tenant_number (tenant_id, number)
);

-- v5 新增: 自动打标规则
CREATE TABLE auto_tag_rules (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  tag_id          BIGINT UNSIGNED NOT NULL,
  conditions      JSON NOT NULL COMMENT '[{"metric":"call_count|duration_sec","operator":">|<|=","value":5,"days":7}]',
  match_mode      ENUM('all','any') NOT NULL DEFAULT 'all',
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_atr_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_atr_tag FOREIGN KEY (tag_id) REFERENCES call_tags(id)
);

-- DNC (Do Not Call) 合规列表
CREATE TABLE dnc_list (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  phone_number    VARCHAR(32) NOT NULL,
  reason          VARCHAR(255),
  source          ENUM('manual','api','customer_request','regulatory') NOT NULL DEFAULT 'manual',
  added_by        BIGINT UNSIGNED,
  expires_at      TIMESTAMP NULL,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_tenant_phone (tenant_id, phone_number),
  CONSTRAINT fk_dnc_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- v5 增强: 新增 ivr_duration_sec, ring_duration_sec, queue_duration_sec, wait_duration_sec
CREATE TABLE calls (
  id                 BIGINT UNSIGNED PRIMARY KEY,
  tenant_id          BIGINT UNSIGNED NOT NULL,
  call_type          ENUM('INBOUND','OUTBOUND','INTERNAL','BACK2BACK','PREDICTIVE','PREVIEW','PROGRESSIVE','POWER','CONFERENCE','CONSULTANT','MONITOR','COACH','BARGE','INTERCEPT','PRIVACY_DIAL','CALLBACK') NOT NULL,
  media_type         ENUM('AUDIO','VIDEO') NOT NULL DEFAULT 'AUDIO' COMMENT '媒体类型（预留视频通话扩展）',
  cli                VARCHAR(32),
  callee             VARCHAR(32),
  masked_callee      VARCHAR(32) NULL COMMENT '脱敏后被叫号码（坐席端显示）',
  did                VARCHAR(32),
  agent_user_id      BIGINT UNSIGNED,
  skill_group_id     BIGINT UNSIGNED,
  ingress_trunk_id   BIGINT UNSIGNED,
  egress_trunk_id    BIGINT UNSIGNED,
  ivr_flow_id        BIGINT UNSIGNED,
  campaign_id        BIGINT UNSIGNED,
  campaign_case_id   BIGINT UNSIGNED COMMENT '所属活动案例',
  parent_call_id     BIGINT UNSIGNED,
  transfer_type      ENUM('blind','attended') NULL,
  hangup_by          ENUM('agent','customer','system') NULL,
  hangup_cause       VARCHAR(64),
  start_at           TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  answer_at          TIMESTAMP(3) NULL,
  end_at             TIMESTAMP(3) NULL,
  duration_sec       INT UNSIGNED GENERATED ALWAYS AS (TIMESTAMPDIFF(SECOND, answer_at, end_at)) VIRTUAL,
  ivr_duration_sec   INT UNSIGNED NULL COMMENT 'IVR 阶段时长(秒)',
  ring_duration_sec  INT UNSIGNED NULL COMMENT '振铃时长(秒)',
  queue_duration_sec INT UNSIGNED NULL COMMENT '排队时长(秒)',
  wait_duration_sec  INT UNSIGNED NULL COMMENT '等待时长(转人工到接通,秒)',
  recording_url      VARCHAR(512),
  realtime_transcript_url VARCHAR(512) COMMENT '实时转写文本存储路径',
  ai_satisfaction_score   TINYINT UNSIGNED COMMENT 'AI 预测满意度 (1-5)',
  ai_session_tags         JSON COMMENT 'AI 自动标签 [{tag, confidence}]',
  ai_summary_mode         ENUM('manual','auto') NULL COMMENT '摘要生成方式',
  ai_summary              TEXT COMMENT 'AI 生成的通话摘要',
  ai_ivr_analysis         JSON COMMENT 'IVR 路径 AI 分析结果（坐席接起前自动分析用户IVR路径与关键信息）',
  ai_completion_score     TINYINT UNSIGNED COMMENT '用户诉求完成度 (1-5, 大模型判断)',
  ai_post_actions         JSON COMMENT '话后处理动作清单（AI自动抽取）',
  disposition_code   VARCHAR(64),
  notes              TEXT,
  csat_score         TINYINT UNSIGNED,
  csat_channel       ENUM('ivr','sms') NULL,
  privacy_mode       BOOLEAN NOT NULL DEFAULT FALSE,
  created_at         TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_tenant_time (tenant_id, start_at),
  INDEX idx_agent_time (agent_user_id, start_at),
  INDEX idx_skill_time (skill_group_id, start_at),
  INDEX idx_campaign (campaign_id, start_at),
  INDEX idx_parent (parent_call_id)
);

CREATE TABLE call_events (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  call_id         BIGINT UNSIGNED NOT NULL,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  event_type      VARCHAR(48) NOT NULL,
  payload         JSON,
  occurred_at     TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_call_time (call_id, occurred_at),
  CONSTRAINT fk_ce_call FOREIGN KEY (call_id) REFERENCES calls(id) ON DELETE CASCADE
);

-- v5 新增: IVR 执行轨迹
CREATE TABLE ivr_tracking (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  call_id         BIGINT UNSIGNED NOT NULL,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  ivr_flow_id     BIGINT UNSIGNED NOT NULL,
  ivr_flow_name   VARCHAR(128),
  node_id         VARCHAR(64) NOT NULL,
  node_type       VARCHAR(48) NOT NULL,
  node_name       VARCHAR(128),
  node_variables  JSON,
  node_exit_code  VARCHAR(48) COMMENT 'Success/Failure/Timeout/Hangup/Default/Branch-A/Overflow等',
  enter_at        TIMESTAMP(3) NOT NULL,
  leave_at        TIMESTAMP(3) NULL,
  INDEX idx_call (call_id, enter_at),
  INDEX idx_tenant_time (tenant_id, enter_at),
  CONSTRAINT fk_ivrt_call FOREIGN KEY (call_id) REFERENCES calls(id)
);

-- v5 新增: 通话级别标签
CREATE TABLE call_tag_assignments (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  call_id         BIGINT UNSIGNED NOT NULL,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  tag_id          BIGINT UNSIGNED NOT NULL,
  assigned_by     BIGINT UNSIGNED,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_call (call_id),
  INDEX idx_tenant_tag (tenant_id, tag_id),
  CONSTRAINT fk_cta_call FOREIGN KEY (call_id) REFERENCES calls(id),
  CONSTRAINT fk_cta_tag FOREIGN KEY (tag_id) REFERENCES call_tags(id)
);

CREATE TABLE queue_snapshots (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  skill_group_id  BIGINT UNSIGNED NOT NULL,
  waiting_count   INT UNSIGNED NOT NULL,
  longest_wait_s  INT UNSIGNED NOT NULL,
  agents_idle     INT UNSIGNED NOT NULL,
  agents_busy     INT UNSIGNED NOT NULL,
  captured_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_sg_time (skill_group_id, captured_at)
);

-- 排队回呼请求
CREATE TABLE callback_requests (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  caller_number   VARCHAR(32) NOT NULL,
  skill_group_id  BIGINT UNSIGNED NOT NULL,
  original_call_id BIGINT UNSIGNED,
  callback_call_id BIGINT UNSIGNED,
  priority        INT NOT NULL DEFAULT 10,
  status          ENUM('pending','calling','connected','failed','expired','cancelled') NOT NULL DEFAULT 'pending',
  max_attempts    TINYINT UNSIGNED NOT NULL DEFAULT 3,
  attempt_count   TINYINT UNSIGNED NOT NULL DEFAULT 0,
  expires_at      TIMESTAMP NOT NULL,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_tenant_status (tenant_id, status, created_at),
  INDEX idx_skill_status (skill_group_id, status)
);

CREATE TABLE campaigns (
  id                    BIGINT UNSIGNED PRIMARY KEY,
  tenant_id             BIGINT UNSIGNED NOT NULL,
  name                  VARCHAR(128) NOT NULL,
  skill_group_id        BIGINT UNSIGNED NOT NULL,
  ivr_flow_id           BIGINT UNSIGNED,
  dialing_mode          ENUM('predictive','preview','progressive','power') NOT NULL DEFAULT 'predictive',
  ratio_multiplier      DECIMAL(3,1) DEFAULT 1.5 COMMENT 'Predictive/Power: 每空闲坐席拨出比率',
  max_abandon_rate      DECIMAL(4,2) DEFAULT 3.00 COMMENT 'Predictive: 最大放弃率阈值(%)，超过自动降速',
  preview_timeout_sec   INT UNSIGNED DEFAULT 30 COMMENT 'Preview: 坐席预览超时秒数',
  concurrent_limit      INT UNSIGNED DEFAULT 0 COMMENT '本活动最大并发呼叫数，0=不限',
  max_attempts          TINYINT UNSIGNED NOT NULL DEFAULT 3,
  retry_interval_min    INT UNSIGNED NOT NULL DEFAULT 30,
  caller_numbers        JSON NOT NULL COMMENT '外显号码列表，轮询使用',
  schedule_start        TIME,
  schedule_end          TIME,
  schedule_days         SET('mon','tue','wed','thu','fri','sat','sun') DEFAULT 'mon,tue,wed,thu,fri' COMMENT '允许外呼的星期',
  timezone              VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai' COMMENT '调度时区',
  dnc_check_enabled     BOOLEAN NOT NULL DEFAULT TRUE,
  status                ENUM('draft','running','paused','completed','aborted') NOT NULL DEFAULT 'draft',
  created_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_camp_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_camp_sg FOREIGN KEY (skill_group_id) REFERENCES skill_groups(id)
);

CREATE TABLE campaign_cases (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  campaign_id     BIGINT UNSIGNED NOT NULL,
  phone_number    VARCHAR(32) NOT NULL,
  customer_name   VARCHAR(128),
  custom_data     JSON,
  attempt_count   TINYINT UNSIGNED NOT NULL DEFAULT 0,
  last_attempt_at TIMESTAMP NULL,
  status          ENUM('pending','calling','connected','failed','completed','skipped','dnc_blocked') NOT NULL DEFAULT 'pending',
  fail_reason     VARCHAR(128),
  call_id         BIGINT UNSIGNED,
  agent_user_id   BIGINT UNSIGNED COMMENT '处理坐席',
  duration_sec    INT UNSIGNED COMMENT '通话时长(秒)',
  disposition_code VARCHAR(64) COMMENT '结案代码',
  next_attempt_at TIMESTAMP NULL COMMENT '下次重拨时间',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_campaign_status (campaign_id, status),
  CONSTRAINT fk_cc_campaign FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE TABLE customers (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128),
  email           VARCHAR(255),
  company         VARCHAR(128),
  level           ENUM('normal','vip','blacklist') NOT NULL DEFAULT 'normal',
  source          VARCHAR(64),
  custom_fields   JSON,
  remark          TEXT,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_tenant_name (tenant_id, name),
  CONSTRAINT fk_cust_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- 客户多号码关联表
CREATE TABLE customer_phones (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  customer_id     BIGINT UNSIGNED NOT NULL,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  phone           VARCHAR(32) NOT NULL,
  label           ENUM('mobile','landline','work','other') NOT NULL DEFAULT 'mobile',
  is_primary      BOOLEAN NOT NULL DEFAULT FALSE,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_tenant_phone (tenant_id, phone),
  CONSTRAINT fk_cp_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE
);

CREATE TABLE customer_interactions (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  customer_id     BIGINT UNSIGNED NOT NULL,
  type            ENUM('call','chat','ticket','email','note') NOT NULL,
  reference_id    BIGINT UNSIGNED,
  agent_user_id   BIGINT UNSIGNED,
  summary         TEXT,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_customer_time (customer_id, created_at),
  INDEX idx_tenant_time (tenant_id, created_at),
  CONSTRAINT fk_ci_customer FOREIGN KEY (customer_id) REFERENCES customers(id)
);

-- v5 新增: 工单类目
CREATE TABLE ticket_categories (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  parent_id       BIGINT UNSIGNED,
  name            VARCHAR(128) NOT NULL,
  sort_order      INT NOT NULL DEFAULT 0,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_tc2_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_tc2_parent FOREIGN KEY (parent_id) REFERENCES ticket_categories(id)
);

-- v5 增强: 新增 category_id, flow_graph, online_status
CREATE TABLE ticket_templates (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128) NOT NULL,
  category_id     BIGINT UNSIGNED COMMENT '所属类目',
  fields_schema   JSON NOT NULL,
  flow_graph      JSON COMMENT '工单流程节点(开始→流程节点→结束，画布式配置)',
  online_status   ENUM('online','offline') NOT NULL DEFAULT 'offline',
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_tt_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_tt_category FOREIGN KEY (category_id) REFERENCES ticket_categories(id)
);

CREATE TABLE tickets (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  template_id     BIGINT UNSIGNED,
  title           VARCHAR(255) NOT NULL,
  description     TEXT,
  priority        ENUM('low','medium','high','urgent') NOT NULL DEFAULT 'medium',
  status          ENUM('open','in_progress','pending','resolved','closed') NOT NULL DEFAULT 'open',
  customer_id     BIGINT UNSIGNED,
  assigned_to     BIGINT UNSIGNED,
  call_id         BIGINT UNSIGNED,
  session_id      BIGINT UNSIGNED,
  custom_fields   JSON,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  resolved_at     TIMESTAMP NULL,
  INDEX idx_tenant_status (tenant_id, status),
  INDEX idx_assigned (assigned_to, status),
  INDEX idx_customer (customer_id),
  CONSTRAINT fk_tk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE ticket_comments (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  ticket_id       BIGINT UNSIGNED NOT NULL,
  author_id       BIGINT UNSIGNED NOT NULL,
  content         TEXT NOT NULL,
  is_internal     BOOLEAN NOT NULL DEFAULT FALSE,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_ticket (ticket_id, created_at),
  CONSTRAINT fk_tc_ticket FOREIGN KEY (ticket_id) REFERENCES tickets(id) ON DELETE CASCADE
);

CREATE TABLE knowledge_categories (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  parent_id       BIGINT UNSIGNED,
  name            VARCHAR(128) NOT NULL,
  sort_order      INT NOT NULL DEFAULT 0,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_kc_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_kc_parent FOREIGN KEY (parent_id) REFERENCES knowledge_categories(id)
);

CREATE TABLE knowledge_articles (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  category_id     BIGINT UNSIGNED,
  title           VARCHAR(255) NOT NULL,
  content         MEDIUMTEXT NOT NULL,
  tags            JSON,
  view_count      INT UNSIGNED NOT NULL DEFAULT 0,
  status          ENUM('draft','published','archived') NOT NULL DEFAULT 'draft',
  created_by      BIGINT UNSIGNED,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_tenant_status (tenant_id, status),
  FULLTEXT idx_ft_title_content (title, content),
  CONSTRAINT fk_ka_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_ka_category FOREIGN KEY (category_id) REFERENCES knowledge_categories(id)
);

CREATE TABLE agent_scripts (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128) NOT NULL,
  script_type     ENUM('inbound','outbound','campaign','general') NOT NULL DEFAULT 'general',
  content         JSON NOT NULL,
  skill_group_id  BIGINT UNSIGNED,
  campaign_id     BIGINT UNSIGNED,
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_as_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE im_channels (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  channel_type    ENUM('web_widget','app','mini_program','dingtalk','email','api') NOT NULL,
  name            VARCHAR(128) NOT NULL,
  config          JSON,
  skill_group_id  BIGINT UNSIGNED,
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_imc_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE im_sessions (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  channel_id      BIGINT UNSIGNED NOT NULL,
  visitor_id      VARCHAR(128),
  customer_id     BIGINT UNSIGNED,
  agent_user_id   BIGINT UNSIGNED,
  skill_group_id  BIGINT UNSIGNED,
  status          ENUM('waiting','active','transferred','closed') NOT NULL DEFAULT 'waiting',
  csat_score      TINYINT UNSIGNED,
  start_at        TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  end_at          TIMESTAMP(3) NULL,
  created_at      TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_tenant_time (tenant_id, start_at),
  INDEX idx_agent (agent_user_id, start_at),
  CONSTRAINT fk_ims_channel FOREIGN KEY (channel_id) REFERENCES im_channels(id)
);

CREATE TABLE im_messages (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  session_id      BIGINT UNSIGNED NOT NULL,
  sender_type     ENUM('visitor','agent','system','bot') NOT NULL,
  sender_id       VARCHAR(128),
  content_type    ENUM('text','image','file','audio','video','card','system') NOT NULL DEFAULT 'text',
  content         TEXT NOT NULL,
  created_at      TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_session_time (session_id, created_at),
  CONSTRAINT fk_imm_session FOREIGN KEY (session_id) REFERENCES im_sessions(id)
);

CREATE TABLE business_hours (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128) NOT NULL,
  timezone        VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
  is_default      BOOLEAN NOT NULL DEFAULT FALSE,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_bh_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE business_hours_schedule (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  business_hours_id BIGINT UNSIGNED NOT NULL,
  day_of_week     TINYINT UNSIGNED,
  specific_date   DATE,
  start_time      TIME NOT NULL,
  end_time        TIME NOT NULL,
  is_holiday      BOOLEAN NOT NULL DEFAULT FALSE,
  remark          VARCHAR(128),
  CONSTRAINT fk_bhs_bh FOREIGN KEY (business_hours_id) REFERENCES business_hours(id) ON DELETE CASCADE
);

CREATE TABLE digital_employees (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128) NOT NULL,
  description     VARCHAR(512),
  asr_provider    ENUM('aliyun_tongyi','iflytek','custom') NOT NULL DEFAULT 'aliyun_tongyi',
  tts_provider    ENUM('aliyun_tongyi','iflytek','custom') NOT NULL DEFAULT 'aliyun_tongyi',
  asr_config      JSON,
  tts_config      JSON,
  dialog_config   JSON,
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_de_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE digital_employee_scenes (
  id              BIGINT UNSIGNED PRIMARY KEY,
  digital_employee_id BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128) NOT NULL,
  intent_config   JSON NOT NULL,
  faq_config      JSON,
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_des_de FOREIGN KEY (digital_employee_id) REFERENCES digital_employees(id)
);

CREATE TABLE recordings (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  call_id         BIGINT UNSIGNED NOT NULL,
  file_path       VARCHAR(512) NOT NULL,
  storage_tier    ENUM('local','minio') NOT NULL DEFAULT 'local',
  file_size       BIGINT UNSIGNED,
  duration_ms     INT UNSIGNED,
  format          ENUM('wav','mp3','ogg') NOT NULL DEFAULT 'wav',
  channels        TINYINT UNSIGNED NOT NULL DEFAULT 2,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at      TIMESTAMP NULL,
  INDEX idx_call (call_id),
  INDEX idx_tenant_time (tenant_id, created_at),
  INDEX idx_expires (expires_at),
  CONSTRAINT fk_rec_call FOREIGN KEY (call_id) REFERENCES calls(id)
);

-- v5 增强: 新增 mailbox_name
CREATE TABLE voicemails (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  call_id         BIGINT UNSIGNED,
  caller_number   VARCHAR(32) NOT NULL,
  skill_group_id  BIGINT UNSIGNED,
  agent_user_id   BIGINT UNSIGNED,
  mailbox_name    VARCHAR(128) COMMENT '语音信箱名称',
  file_path       VARCHAR(512) NOT NULL,
  duration_ms     INT UNSIGNED,
  is_read         BOOLEAN NOT NULL DEFAULT FALSE,
  read_at         TIMESTAMP NULL,
  read_by         BIGINT UNSIGNED,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_tenant_unread (tenant_id, is_read, created_at),
  INDEX idx_agent (agent_user_id, is_read)
);

CREATE TABLE csat_configs (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128) NOT NULL,
  method          ENUM('ivr','sms') NOT NULL,
  ivr_flow_id     BIGINT UNSIGNED,
  sms_template    TEXT,
  score_range     TINYINT UNSIGNED NOT NULL DEFAULT 5,
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_csat_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE csat_results (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  call_id         BIGINT UNSIGNED,
  session_id      BIGINT UNSIGNED,
  agent_user_id   BIGINT UNSIGNED,
  score           TINYINT UNSIGNED NOT NULL,
  channel         ENUM('ivr','sms','web') NOT NULL,
  comment         TEXT,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_tenant_time (tenant_id, created_at),
  INDEX idx_agent (agent_user_id, created_at)
);

CREATE TABLE session_info_templates (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128) NOT NULL,
  fields_schema   JSON NOT NULL,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_sit_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- v5 新增: 自定义字段元数据（客户/会话信息/工单共用）
CREATE TABLE custom_field_definitions (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  scope           ENUM('customer','session_info','ticket') NOT NULL,
  field_name      VARCHAR(128) NOT NULL,
  field_key       VARCHAR(64) NOT NULL,
  field_type      ENUM('text','textarea','number','dropdown','checkbox','date','phone','email') NOT NULL,
  options         JSON COMMENT '下拉框等选项值列表',
  is_required     BOOLEAN NOT NULL DEFAULT FALSE,
  is_system       BOOLEAN NOT NULL DEFAULT FALSE,
  sort_order      INT NOT NULL DEFAULT 0,
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_tenant_scope_key (tenant_id, scope, field_key),
  CONSTRAINT fk_cfd_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- v5 新增: 来电弹屏配置
CREATE TABLE screen_pop_configs (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128) NOT NULL,
  url             VARCHAR(512) NOT NULL COMMENT '仅 HTTPS',
  is_home         BOOLEAN NOT NULL DEFAULT FALSE COMMENT '是否为首页弹屏',
  sort_order      INT NOT NULL DEFAULT 0,
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_spc_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);
-- 限制: 每个租户最多 5 个弹屏页面

CREATE TABLE webhook_configs (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128) NOT NULL,
  url             VARCHAR(512) NOT NULL,
  method          ENUM('POST','PUT') NOT NULL DEFAULT 'POST',
  headers         JSON,
  secret          VARCHAR(255),
  event_types     JSON NOT NULL,
  retry_count     TINYINT UNSIGNED NOT NULL DEFAULT 3,
  retry_interval_sec INT UNSIGNED NOT NULL DEFAULT 60,
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_wh_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE webhook_delivery_log (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  webhook_id      BIGINT UNSIGNED NOT NULL,
  event_type      VARCHAR(48) NOT NULL,
  payload         JSON NOT NULL,
  response_status SMALLINT,
  response_body   TEXT,
  attempt         TINYINT UNSIGNED NOT NULL DEFAULT 1,
  delivered_at    TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_webhook_time (webhook_id, delivered_at)
);

CREATE TABLE audit_logs (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  user_id         BIGINT UNSIGNED,
  action          VARCHAR(64) NOT NULL,
  resource_type   VARCHAR(64) NOT NULL,
  resource_id     BIGINT UNSIGNED,
  detail          JSON,
  ip_address      VARCHAR(45),
  user_agent      VARCHAR(512),
  created_at      TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_tenant_time (tenant_id, created_at),
  INDEX idx_user_time (user_id, created_at),
  INDEX idx_resource (resource_type, resource_id)
);

CREATE TABLE performance_scorecards (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  user_id         BIGINT UNSIGNED NOT NULL,
  period_start    DATE NOT NULL,
  period_end      DATE NOT NULL,
  metrics         JSON NOT NULL,
  total_score     DECIMAL(5,2),
  rank            INT UNSIGNED,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_tenant_period (tenant_id, period_start),
  INDEX idx_user_period (user_id, period_start)
);

-- v5.1 AI 增强: 质检规则（支持规则质检 + 大模型质检）
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
  INDEX idx_tenant (tenant_id),
  CONSTRAINT fk_qr_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- 质检方案（多规则组合）
CREATE TABLE qa_schemes (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128) NOT NULL,
  base_score      INT NOT NULL DEFAULT 100 COMMENT '基础分',
  is_template     BOOLEAN NOT NULL DEFAULT FALSE COMMENT '行业预置模板',
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_tenant (tenant_id),
  CONSTRAINT fk_qs_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- 质检方案-规则关联
CREATE TABLE qa_scheme_rules (
  scheme_id       BIGINT UNSIGNED NOT NULL,
  rule_id         BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (scheme_id, rule_id),
  CONSTRAINT fk_qsr_scheme FOREIGN KEY (scheme_id) REFERENCES qa_schemes(id) ON DELETE CASCADE,
  CONSTRAINT fk_qsr_rule FOREIGN KEY (rule_id) REFERENCES qa_rules(id)
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
  appeal_reason   TEXT COMMENT '申诉原因',
  appeal_result   TEXT COMMENT '申诉处理结果',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_tenant_call (tenant_id, call_id),
  INDEX idx_tenant_time (tenant_id, created_at),
  INDEX idx_review (tenant_id, review_status)
);

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
  INDEX idx_tenant_scope (tenant_id, scope, scope_id),
  CONSTRAINT fk_qrp_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE TABLE asr_hotwords (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128) NOT NULL,
  words           JSON NOT NULL COMMENT '["词1","词2",...]',
  status          ENUM('active','disabled') NOT NULL DEFAULT 'active',
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_tenant (tenant_id),
  CONSTRAINT fk_ah_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- 数字员工训练闭环：标注任务
CREATE TABLE annotation_tasks (
  id              BIGINT UNSIGNED PRIMARY KEY,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  name            VARCHAR(128) NOT NULL,
  digital_employee_id BIGINT UNSIGNED NOT NULL,
  status          ENUM('in_progress','completed','closed') NOT NULL DEFAULT 'in_progress',
  sample_mode     ENUM('full','ratio','custom') NOT NULL DEFAULT 'full' COMMENT '抽样模式',
  sample_value    INT UNSIGNED NULL COMMENT '比例或自定义数量',
  total_count     INT UNSIGNED NOT NULL DEFAULT 0,
  completed_count INT UNSIGNED NOT NULL DEFAULT 0,
  filter_start_at TIMESTAMP NULL COMMENT '通话时间筛选起始',
  filter_end_at   TIMESTAMP NULL COMMENT '通话时间筛选结束',
  filter_exclude_annotated BOOLEAN NOT NULL DEFAULT FALSE,
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_tenant (tenant_id),
  CONSTRAINT fk_at_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- 标注结果
CREATE TABLE annotation_results (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  task_id         BIGINT UNSIGNED NOT NULL,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  call_id         BIGINT UNSIGNED NOT NULL,
  turn_index      INT UNSIGNED NOT NULL COMMENT '对话轮次序号',
  asr_result      VARCHAR(64) COMMENT '语义识别结果标注(correct/wrong/partial/no_response等)',
  intent_label    VARCHAR(128) COMMENT '意图标注',
  correct_text    TEXT COMMENT '正确转译结果',
  hotword_labels  JSON COMMENT '热词标注 [{word, type}]',
  tag_labels      JSON COMMENT '标签标注',
  annotator_id    BIGINT UNSIGNED COMMENT '标注人',
  annotated_at    TIMESTAMP NULL,
  INDEX idx_task (task_id),
  INDEX idx_call (call_id),
  CONSTRAINT fk_ar_task FOREIGN KEY (task_id) REFERENCES annotation_tasks(id) ON DELETE CASCADE
);

CREATE TABLE webrtc_quality_logs (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  call_id         BIGINT UNSIGNED NOT NULL,
  tenant_id       BIGINT UNSIGNED NOT NULL,
  agent_user_id   BIGINT UNSIGNED NOT NULL,
  jitter_ms       FLOAT COMMENT '抖动(毫秒)',
  packet_loss_pct FLOAT COMMENT '丢包率(%)',
  rtt_ms          FLOAT COMMENT '往返延迟(毫秒)',
  mos_score       FLOAT COMMENT 'MOS 评分',
  codec           VARCHAR(16) COMMENT '编解码器(PCMU/OPUS等)',
  local_ip        VARCHAR(64),
  remote_ip       VARCHAR(64),
  sender_report   JSON COMMENT '发送报告原始数据',
  receiver_report JSON COMMENT '接收报告原始数据',
  collected_at    TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  INDEX idx_call (call_id),
  INDEX idx_tenant_time (tenant_id, collected_at)
);

