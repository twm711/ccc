package ticket

import "context"

type TicketCategoryRepository interface {
	Create(ctx context.Context, c *TicketCategory) error
	List(ctx context.Context, tenantID int64) ([]*TicketCategory, error)
	GetByID(ctx context.Context, id int64) (*TicketCategory, error)
}

type TicketTemplateRepository interface {
	Create(ctx context.Context, t *TicketTemplate) error
	GetByID(ctx context.Context, id int64) (*TicketTemplate, error)
	Update(ctx context.Context, t *TicketTemplate) error
	List(ctx context.Context, tenantID int64) ([]*TicketTemplate, error)
}

type TicketRepository interface {
	Create(ctx context.Context, t *Ticket) error
	GetByID(ctx context.Context, id int64) (*Ticket, error)
	Update(ctx context.Context, t *Ticket) error
	List(ctx context.Context, tenantID int64, offset, limit int) ([]*Ticket, error)
	ListByCallID(ctx context.Context, callID int64) ([]*Ticket, error)
}

type TicketCommentRepository interface {
	Create(ctx context.Context, c *TicketComment) error
	ListByTicket(ctx context.Context, ticketID int64) ([]*TicketComment, error)
}
