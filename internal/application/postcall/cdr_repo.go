package postcall

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// CDRRepository upserts per-tenant daily call aggregates.
type CDRRepository interface {
	UpsertDailyCDR(ctx context.Context, e CDREntry) error
}

// CDREntry is a single call's contribution to the daily aggregate.
type CDREntry struct {
	TenantID    int64
	BucketDate  string // YYYY-MM-DD
	Inbound     int
	Outbound    int
	Answered    int
	Abandoned   int
	TalkSeconds int
	RingSeconds int
	QueueSeconds int
}

// MySQLCDRRepo writes to the daily_cdr_summary table.
type MySQLCDRRepo struct {
	db *sqlx.DB
}

func NewMySQLCDRRepo(db *sqlx.DB) *MySQLCDRRepo {
	return &MySQLCDRRepo{db: db}
}

// UpsertDailyCDR adds the entry's counters to the daily summary row, creating it if missing.
func (r *MySQLCDRRepo) UpsertDailyCDR(ctx context.Context, e CDREntry) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO daily_cdr_summary
			(tenant_id, bucket_date, inbound_calls, outbound_calls, answered_calls, abandoned_calls,
			 total_talk_sec, total_ring_sec, total_queue_sec)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			inbound_calls   = inbound_calls   + VALUES(inbound_calls),
			outbound_calls  = outbound_calls  + VALUES(outbound_calls),
			answered_calls  = answered_calls  + VALUES(answered_calls),
			abandoned_calls = abandoned_calls + VALUES(abandoned_calls),
			total_talk_sec  = total_talk_sec  + VALUES(total_talk_sec),
			total_ring_sec  = total_ring_sec  + VALUES(total_ring_sec),
			total_queue_sec = total_queue_sec + VALUES(total_queue_sec)`,
		e.TenantID, e.BucketDate,
		e.Inbound, e.Outbound, e.Answered, e.Abandoned,
		e.TalkSeconds, e.RingSeconds, e.QueueSeconds,
	)
	return err
}
