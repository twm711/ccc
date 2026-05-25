package mysql

import (
	"context"
	"fmt"

	"github.com/divord97/ccc/internal/domain/platform"
	"github.com/divord97/ccc/pkg/snowflake"
	"github.com/rs/zerolog/log"
	"time"
)

// RecordingAccessLogger logs recording access events to the audit_logs table.
type RecordingAccessLogger struct {
	repo platform.AuditLogRepository
}

func NewRecordingAccessLogger(repo platform.AuditLogRepository) *RecordingAccessLogger {
	return &RecordingAccessLogger{repo: repo}
}

func (l *RecordingAccessLogger) LogAccess(ctx context.Context, tenantID, userID, recordingID int64, action, ip string) {
	entry := &platform.AuditLog{
		ID:        snowflake.NextID(),
		TenantID:  tenantID,
		UserID:    userID,
		Action:    fmt.Sprintf("recording.%s", action),
		Resource:  fmt.Sprintf("/recordings/%d", recordingID),
		IP:        ip,
		CreatedAt: time.Now(),
	}
	if err := l.repo.Create(ctx, entry); err != nil {
		log.Error().Err(err).Int64("recording_id", recordingID).Msg("recording access log failed")
	}
}
