package mysql

import (
	"context"
	"database/sql"

	"github.com/divord97/ccc/internal/domain/ticket"
	"github.com/jmoiron/sqlx"
)

type TicketCategoryRepo struct{ db *sqlx.DB }

func NewTicketCategoryRepo(db *sqlx.DB) *TicketCategoryRepo { return &TicketCategoryRepo{db: db} }

func (r *TicketCategoryRepo) Create(ctx context.Context, c *ticket.TicketCategory) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ticket_categories (id, tenant_id, name, parent_id) VALUES (?,?,?,?)`,
		c.ID, c.TenantID, c.Name, c.ParentID)
	return err
}

func (r *TicketCategoryRepo) List(ctx context.Context, tenantID int64) ([]*ticket.TicketCategory, error) {
	var result []*ticket.TicketCategory
	err := r.db.SelectContext(ctx, &result, `SELECT * FROM ticket_categories WHERE tenant_id=?`, tenantID)
	return result, err
}

func (r *TicketCategoryRepo) GetByID(ctx context.Context, id int64) (*ticket.TicketCategory, error) {
	var c ticket.TicketCategory
	if err := r.db.GetContext(ctx, &c, `SELECT * FROM ticket_categories WHERE id=?`, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

type TicketTemplateRepo struct{ db *sqlx.DB }

func NewTicketTemplateRepo(db *sqlx.DB) *TicketTemplateRepo { return &TicketTemplateRepo{db: db} }

func (r *TicketTemplateRepo) Create(ctx context.Context, t *ticket.TicketTemplate) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ticket_templates (id, tenant_id, name, category_id, fields, flow_graph, online_status, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		t.ID, t.TenantID, t.Name, t.CategoryID, t.Fields, t.FlowGraph, t.OnlineStatus, t.CreatedAt, t.UpdatedAt)
	return err
}

func (r *TicketTemplateRepo) GetByID(ctx context.Context, id int64) (*ticket.TicketTemplate, error) {
	var t ticket.TicketTemplate
	if err := r.db.GetContext(ctx, &t, `SELECT * FROM ticket_templates WHERE id=?`, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *TicketTemplateRepo) Update(ctx context.Context, t *ticket.TicketTemplate) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ticket_templates SET name=?, category_id=?, fields=?, flow_graph=?, online_status=?, updated_at=? WHERE id=?`,
		t.Name, t.CategoryID, t.Fields, t.FlowGraph, t.OnlineStatus, t.UpdatedAt, t.ID)
	return err
}

func (r *TicketTemplateRepo) List(ctx context.Context, tenantID int64) ([]*ticket.TicketTemplate, error) {
	var result []*ticket.TicketTemplate
	err := r.db.SelectContext(ctx, &result, `SELECT * FROM ticket_templates WHERE tenant_id=?`, tenantID)
	return result, err
}

type TicketRepo struct{ db *sqlx.DB }

func NewTicketRepo(db *sqlx.DB) *TicketRepo { return &TicketRepo{db: db} }

func (r *TicketRepo) Create(ctx context.Context, t *ticket.Ticket) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tickets (id, tenant_id, template_id, category_id, title, description, status, priority,
		 customer_id, assignee_id, call_id, custom_data, created_at, updated_at, resolved_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.TenantID, t.TemplateID, t.CategoryID, t.Title, t.Description, t.Status, t.Priority,
		t.CustomerID, t.AssigneeID, t.CallID, t.CustomData, t.CreatedAt, t.UpdatedAt, t.ResolvedAt)
	return err
}

func (r *TicketRepo) GetByID(ctx context.Context, id int64) (*ticket.Ticket, error) {
	var t ticket.Ticket
	if err := r.db.GetContext(ctx, &t, `SELECT * FROM tickets WHERE id=?`, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *TicketRepo) Update(ctx context.Context, t *ticket.Ticket) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tickets SET title=?, description=?, status=?, priority=?, assignee_id=?, custom_data=?, updated_at=?, resolved_at=? WHERE id=?`,
		t.Title, t.Description, t.Status, t.Priority, t.AssigneeID, t.CustomData, t.UpdatedAt, t.ResolvedAt, t.ID)
	return err
}

func (r *TicketRepo) List(ctx context.Context, tenantID int64, offset, limit int) ([]*ticket.Ticket, error) {
	var result []*ticket.Ticket
	err := r.db.SelectContext(ctx, &result,
		`SELECT * FROM tickets WHERE tenant_id=? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		tenantID, limit, offset)
	return result, err
}

func (r *TicketRepo) ListByCallID(ctx context.Context, callID int64) ([]*ticket.Ticket, error) {
	var result []*ticket.Ticket
	err := r.db.SelectContext(ctx, &result,
		`SELECT * FROM tickets WHERE call_id=? ORDER BY created_at DESC`, callID)
	return result, err
}

type TicketCommentRepo struct{ db *sqlx.DB }

func NewTicketCommentRepo(db *sqlx.DB) *TicketCommentRepo { return &TicketCommentRepo{db: db} }

func (r *TicketCommentRepo) Create(ctx context.Context, c *ticket.TicketComment) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO ticket_comments (id, ticket_id, author_id, content, created_at) VALUES (?,?,?,?,?)`,
		c.ID, c.TicketID, c.AuthorID, c.Content, c.CreatedAt)
	return err
}

func (r *TicketCommentRepo) ListByTicket(ctx context.Context, ticketID int64) ([]*ticket.TicketComment, error) {
	var result []*ticket.TicketComment
	err := r.db.SelectContext(ctx, &result,
		`SELECT * FROM ticket_comments WHERE ticket_id=? ORDER BY created_at`, ticketID)
	return result, err
}
