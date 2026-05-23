package im

import "time"

type ChannelType string

const (
	ChannelTypeWebWidget  ChannelType = "web_widget"
	ChannelTypeApp        ChannelType = "app"
	ChannelTypeMiniProg   ChannelType = "mini_program"
	ChannelTypeDingTalk   ChannelType = "dingtalk"
	ChannelTypeEmail      ChannelType = "email"
	ChannelTypeAPI        ChannelType = "api"
)

type ChannelStatus string

const (
	ChannelStatusActive   ChannelStatus = "active"
	ChannelStatusDisabled ChannelStatus = "disabled"
)

type SessionStatus string

const (
	SessionStatusWaiting     SessionStatus = "waiting"
	SessionStatusActive      SessionStatus = "active"
	SessionStatusTransferred SessionStatus = "transferred"
	SessionStatusClosed      SessionStatus = "closed"
)

type SenderType string

const (
	SenderTypeVisitor SenderType = "visitor"
	SenderTypeAgent   SenderType = "agent"
	SenderTypeSystem  SenderType = "system"
	SenderTypeBot     SenderType = "bot"
)

type ContentType string

const (
	ContentTypeText   ContentType = "text"
	ContentTypeImage  ContentType = "image"
	ContentTypeFile   ContentType = "file"
	ContentTypeAudio  ContentType = "audio"
	ContentTypeVideo  ContentType = "video"
	ContentTypeCard   ContentType = "card"
	ContentTypeSystem ContentType = "system"
)

type IMChannel struct {
	ID           int64         `db:"id" json:"id"`
	TenantID     int64         `db:"tenant_id" json:"tenant_id"`
	ChannelType  ChannelType   `db:"channel_type" json:"channel_type"`
	Name         string        `db:"name" json:"name"`
	Config       string        `db:"config" json:"config"`
	SkillGroupID *int64        `db:"skill_group_id" json:"skill_group_id"`
	Status       ChannelStatus `db:"status" json:"status"`
	CreatedAt    time.Time     `db:"created_at" json:"created_at"`
}

type IMSession struct {
	ID           int64         `db:"id" json:"id"`
	TenantID     int64         `db:"tenant_id" json:"tenant_id"`
	ChannelID    int64         `db:"channel_id" json:"channel_id"`
	VisitorID    string        `db:"visitor_id" json:"visitor_id"`
	CustomerID   *int64        `db:"customer_id" json:"customer_id"`
	AgentUserID  *int64        `db:"agent_user_id" json:"agent_user_id"`
	SkillGroupID *int64        `db:"skill_group_id" json:"skill_group_id"`
	Status       SessionStatus `db:"status" json:"status"`
	CSATScore    *int          `db:"csat_score" json:"csat_score"`
	StartAt      time.Time     `db:"start_at" json:"start_at"`
	EndAt        *time.Time    `db:"end_at" json:"end_at"`
	CreatedAt    time.Time     `db:"created_at" json:"created_at"`
}

type IMMessage struct {
	ID          int64       `db:"id" json:"id"`
	SessionID   int64       `db:"session_id" json:"session_id"`
	SenderType  SenderType  `db:"sender_type" json:"sender_type"`
	SenderID    string      `db:"sender_id" json:"sender_id"`
	ContentType ContentType `db:"content_type" json:"content_type"`
	Content     string      `db:"content" json:"content"`
	CreatedAt   time.Time   `db:"created_at" json:"created_at"`
}
