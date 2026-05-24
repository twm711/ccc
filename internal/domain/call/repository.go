package call

import (
	"context"
	"time"
)

type CallListFilter struct {
	Direction *CallDirection
	CallType  *CallType
	MediaType *MediaType
	Status    *CallStatus
	Caller    string
	Callee    string
	StartFrom *time.Time
	StartTo   *time.Time
}

type CallRepository interface {
	Create(ctx context.Context, c *Call) error
	GetByID(ctx context.Context, id int64) (*Call, error)
	Update(ctx context.Context, c *Call) error
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*Call, int64, error)
	ListWithFilter(ctx context.Context, tenantID int64, filter CallListFilter, offset, limit int) ([]*Call, int64, error)
}

type CallEventRepository interface {
	Create(ctx context.Context, e *CallEvent) error
	ListByCallID(ctx context.Context, callID int64) ([]*CallEvent, error)
}

type IVRTrackingRepository interface {
	Create(ctx context.Context, t *IVRTracking) error
	ListByCallID(ctx context.Context, callID int64) ([]*IVRTracking, error)
}

type RecordingRepository interface {
	Create(ctx context.Context, r *Recording) error
	GetByID(ctx context.Context, id int64) (*Recording, error)
	GetByCallID(ctx context.Context, callID int64) (*Recording, error)
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*Recording, int64, error)
}

type QueueSnapshotRepository interface {
	Create(ctx context.Context, s *QueueSnapshot) error
	GetLatest(ctx context.Context, tenantID, skillGroupID int64) (*QueueSnapshot, error)
}

type VoicemailRepository interface {
	Create(ctx context.Context, v *Voicemail) error
	GetByID(ctx context.Context, id int64) (*Voicemail, error)
	Update(ctx context.Context, v *Voicemail) error
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*Voicemail, int64, error)
	Delete(ctx context.Context, id int64) error
}

type CallbackRequestRepository interface {
	Create(ctx context.Context, r *CallbackRequest) error
	GetByID(ctx context.Context, id int64) (*CallbackRequest, error)
	Update(ctx context.Context, r *CallbackRequest) error
	ListPending(ctx context.Context, tenantID int64) ([]*CallbackRequest, error)
}
