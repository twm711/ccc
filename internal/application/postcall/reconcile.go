package postcall

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

// Reconciler rebuilds daily_cdr_summary from the source-of-truth calls table.
// Runs once per day to backfill any rows that NATS post-call processing missed
// (consumer outage, message loss, schema drift). Idempotent via cdr_processed_calls.
type Reconciler struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

func NewReconciler(db *sqlx.DB, logger zerolog.Logger) *Reconciler {
	return &Reconciler{db: db, logger: logger}
}

// Run blocks until ctx is canceled, executing reconcile once per day at the
// configured hour (UTC). Initial run happens immediately so that fresh
// deployments don't wait 24h to backfill.
func (r *Reconciler) Run(ctx context.Context, runHourUTC int) {
	if runHourUTC < 0 || runHourUTC > 23 {
		runHourUTC = 2
	}
	r.reconcileYesterday(ctx)
	for {
		next := nextDailyRun(time.Now().UTC(), runHourUTC)
		t := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
			r.reconcileYesterday(ctx)
		}
	}
}

func nextDailyRun(now time.Time, hour int) time.Time {
	candidate := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if !candidate.After(now) {
		candidate = candidate.Add(24 * time.Hour)
	}
	return candidate
}

func (r *Reconciler) reconcileYesterday(ctx context.Context) {
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	// Pull calls that ended on the target day but haven't been folded into the
	// dedup table. Mark them processed and increment the daily summary in one
	// transaction per call so a partial run can resume.
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id, c.tenant_id, c.direction, c.duration_sec, c.ring_duration_sec,
		       c.queue_duration_sec, c.answered_at IS NOT NULL AS answered
		FROM calls c
		LEFT JOIN cdr_processed_calls p ON p.call_id = c.id
		WHERE DATE(c.started_at) = ? AND p.call_id IS NULL AND c.ended_at IS NOT NULL`,
		yesterday)
	if err != nil {
		r.logger.Error().Err(err).Str("bucket", yesterday).Msg("postcall reconcile: query missing calls failed")
		return
	}
	defer rows.Close()

	repo := NewMySQLCDRRepo(r.db)
	repaired := 0
	for rows.Next() {
		var (
			callID, tenantID                                 int64
			direction                                        string
			durationSec, ringDurationSec, queueDurationSec   int
			answered                                         bool
		)
		if err := rows.Scan(&callID, &tenantID, &direction, &durationSec, &ringDurationSec, &queueDurationSec, &answered); err != nil {
			r.logger.Error().Err(err).Msg("postcall reconcile: row scan failed")
			continue
		}
		entry := CDREntry{
			CallID:       callID,
			TenantID:     tenantID,
			BucketDate:   yesterday,
			TalkSeconds:  durationSec,
			RingSeconds:  ringDurationSec,
			QueueSeconds: queueDurationSec,
		}
		if direction == "inbound" {
			entry.Inbound = 1
		} else {
			entry.Outbound = 1
		}
		if answered {
			entry.Answered = 1
		} else {
			entry.Abandoned = 1
		}
		if err := repo.UpsertDailyCDR(ctx, entry); err != nil {
			r.logger.Error().Err(err).Int64("call_id", callID).Msg("postcall reconcile: upsert failed")
			continue
		}
		repaired++
	}
	if err := rows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("postcall reconcile: rows iteration error")
	}
	if repaired > 0 {
		r.logger.Info().Int("repaired", repaired).Str("bucket", yesterday).Msg("postcall reconcile: backfilled missing CDR rows")
	}
}
