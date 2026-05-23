package telephony

import "context"

type CarrierRepository interface {
	Create(ctx context.Context, c *Carrier) error
	GetByID(ctx context.Context, id int64) (*Carrier, error)
	Update(ctx context.Context, c *Carrier) error
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*Carrier, int64, error)
}

type SIPTrunkRepository interface {
	Create(ctx context.Context, t *SIPTrunk) error
	GetByID(ctx context.Context, id int64) (*SIPTrunk, error)
	Update(ctx context.Context, t *SIPTrunk) error
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*SIPTrunk, int64, error)
}

type PhoneNumberRepository interface {
	Create(ctx context.Context, p *PhoneNumber) error
	GetByID(ctx context.Context, id int64) (*PhoneNumber, error)
	GetByNumber(ctx context.Context, tenantID int64, number string) (*PhoneNumber, error)
	Update(ctx context.Context, p *PhoneNumber) error
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*PhoneNumber, int64, error)
}

type CallNumberTagRepository interface {
	Create(ctx context.Context, t *CallNumberTag) error
	ListByNumber(ctx context.Context, tenantID int64, number string) ([]*CallNumberTag, error)
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*CallNumberTag, int64, error)
	Delete(ctx context.Context, id int64) error
}

type AutoTagRuleRepository interface {
	Create(ctx context.Context, r *AutoTagRule) error
	GetByID(ctx context.Context, id int64) (*AutoTagRule, error)
	Update(ctx context.Context, r *AutoTagRule) error
	ListActive(ctx context.Context, tenantID int64) ([]*AutoTagRule, error)
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*AutoTagRule, int64, error)
	Delete(ctx context.Context, id int64) error
}
