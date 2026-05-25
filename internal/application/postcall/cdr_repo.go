package postcall

import (
	"context"
	"errors"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// CDRRepository upserts per-tenant daily call aggregates with at-most-once semantics.
type CDRRepository interface {
	UpsertDailyCDR(ctx context.Context, e CDREntry) error
}

// CDREntry is a single call's contribution to the daily aggregate.
// CallID is required to ensure NATS redeliveries don't double-count.
type CDREntry struct {
	CallID       int64
	TenantID     int64
	BucketDate   string // YYYY-MM-DD
	Inbound      int
	Outbound     int
	Answered     int
	Abandoned    int
	TalkSeconds  int
	RingSeconds  int
	QueueSeconds int
}

// MySQLCDRRepo writes to the daily_cdr_summary table.
type MySQLCDRRepo struct {
	db *sqlx.DB
}

func NewMySQLCDRRepo(db *sqlx.DB) *MySQLCDRRepo {
	return &MySQLCDRRepo{db: db}
}

// mysqlDuplicateEntry is the MySQL error code for unique-key violations.
const mysqlDuplicateEntry = 1062

// UpsertDailyCDR records this call into the dedup table and adds its counters
// to the daily summary in a single transaction. If the call was already
// processed (duplicate primary key in cdr_processed_calls), the call returns
// nil without touching the aggregate — making redelivery a no-op.
func (r *MySQLCDRRepo) UpsertDailyCDR(ctx context.Context, e CDREntry) error {
	if e.CallID == 0 {
		return errors.New("postcall: CDREntry.CallID is required for idempotent upsert")
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO cdr_processed_calls (call_id) VALUES (?)`, e.CallID); err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateEntry {
			return nil // already processed, treat redelivery as success
		}
		return err
	}

	if _, err := tx.ExecContext(ctx, `
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
	); err != nil {
		return err
	}

	return tx.Commit()
}