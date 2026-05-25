package ticket

import (
	"context"
	"encoding/json"
	"time"

	"github.com/divord97/ccc/pkg/snowflake"
)

var validPriorities = map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}

// validTransitions defines the allowed ticket status state machine.
var validTransitions = map[TicketStatus][]TicketStatus{
	TicketStatusOpen:       {TicketStatusInProgress, TicketStatusPending},
	TicketStatusInProgress: {TicketStatusPending, TicketStatusResolved},
	TicketStatusPending:    {TicketStatusInProgress, TicketStatusResolved},
	TicketStatusResolved:   {TicketStatusClosed, TicketStatusInProgress},
	TicketStatusClosed:     {},
}

// TicketTemplateService manages ticket template lifecycle.
type TicketTemplateService struct {
	templates  TicketTemplateRepository
	categories TicketCategoryRepository
}

func NewTicketTemplateService(templates TicketTemplateRepository, categories TicketCategoryRepository) *TicketTemplateService {
	return &TicketTemplateService{templates: templates, categories: categories}
}

type CreateTemplateInput struct {
	TenantID   int64  `json:"tenant_id"`
	Name       string `json:"name"`
	CategoryID *int64 `json:"category_id"`
	Fields     string `json:"fields"`
	FlowGraph  string `json:"flow_graph"`
}

func (s *TicketTemplateService) Create(ctx context.Context, in CreateTemplateInput) (*TicketTemplate, error) {
	if in.FlowGraph != "" {
		if !json.Valid([]byte(in.FlowGraph)) {
			return nil, ErrInvalidFlowGraph
		}
	}

	now := time.Now()
	t := &TicketTemplate{
		ID:           snowflake.NextID(),
		TenantID:     in.TenantID,
		Name:         in.Name,
		CategoryID:   in.CategoryID,
		Fields:       in.Fields,
		FlowGraph:    in.FlowGraph,
		OnlineStatus: "draft",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.templates.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TicketTemplateService) Publish(ctx context.Context, id int64) (*TicketTemplate, error) {
	t, err := s.templates.GetByID(ctx, id)
	if err != nil || t == nil {
		return nil, ErrTemplateNotFound
	}
	if t.OnlineStatus == "published" {
		return nil, ErrAlreadyPublished
	}

	t.OnlineStatus = "published"
	t.UpdatedAt = time.Now()
	if err := s.templates.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TicketTemplateService) Offline(ctx context.Context, id int64) (*TicketTemplate, error) {
	t, err := s.templates.GetByID(ctx, id)
	if err != nil || t == nil {
		return nil, ErrTemplateNotFound
	}

	t.OnlineStatus = "offline"
	t.UpdatedAt = time.Now()
	if err := s.templates.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TicketTemplateService) GetTemplate(ctx context.Context, id int64) (*TicketTemplate, error) {
	t, err := s.templates.GetByID(ctx, id)
	if err != nil || t == nil {
		return nil, ErrTemplateNotFound
	}
	return t, nil
}

func (s *TicketTemplateService) UpdateTemplate(ctx context.Context, t *TicketTemplate) error {
	t.UpdatedAt = time.Now()
	return s.templates.Update(ctx, t)
}

func (s *TicketTemplateService) ListTemplates(ctx context.Context, tenantID int64) ([]*TicketTemplate, error) {
	return s.templates.List(ctx, tenantID)
}

func (s *TicketTemplateService) CreateCategory(ctx context.Context, tenantID int64, name string, parentID *int64) (*TicketCategory, error) {
	c := &TicketCategory{
		ID:       snowflake.NextID(),
		TenantID: tenantID,
		Name:     name,
		ParentID: parentID,
	}
	if err := s.categories.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *TicketTemplateService) ListCategories(ctx context.Context, tenantID int64) ([]*TicketCategory, error) {
	return s.categories.List(ctx, tenantID)
}

// TicketService manages ticket lifecycle.
type TicketService struct {
	tickets   TicketRepository
	templates TicketTemplateRepository
	comments  TicketCommentRepository
}

func NewTicketService(tickets TicketRepository, templates TicketTemplateRepository, comments TicketCommentRepository) *TicketService {
	return &TicketService{tickets: tickets, templates: templates, comments: comments}
}

type CreateTicketInput struct {
	TenantID    int64  `json:"tenant_id"`
	TemplateID  *int64 `json:"template_id"`
	CategoryID  *int64 `json:"category_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	CustomerID  *int64 `json:"customer_id"`
	CallID      *int64 `json:"call_id"`
	CustomData  string `json:"custom_data"`
}

func (s *TicketService) Create(ctx context.Context, in CreateTicketInput) (*Ticket, error) {
	if !validPriorities[in.Priority] {
		return nil, ErrInvalidPriority
	}

	if in.TemplateID != nil {
		tmpl, err := s.templates.GetByID(ctx, *in.TemplateID)
		if err != nil || tmpl == nil {
			return nil, ErrTemplateNotFound
		}
		if tmpl.OnlineStatus != "published" {
			return nil, ErrTemplateNotPublished
		}
	}

	now := time.Now()
	tk := &Ticket{
		ID:          snowflake.NextID(),
		TenantID:    in.TenantID,
		TemplateID:  in.TemplateID,
		CategoryID:  in.CategoryID,
		Title:       in.Title,
		Description: in.Description,
		Status:      TicketStatusOpen,
		Priority:    in.Priority,
		CustomerID:  in.CustomerID,
		CallID:      in.CallID,
		CustomData:  in.CustomData,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.tickets.Create(ctx, tk); err != nil {
		return nil, err
	}
	return tk, nil
}

func (s *TicketService) GetByID(ctx context.Context, id int64) (*Ticket, error) {
	tk, err := s.tickets.GetByID(ctx, id)
	if err != nil || tk == nil {
		return nil, ErrTicketNotFound
	}
	return tk, nil
}

func (s *TicketService) Update(ctx context.Context, tk *Ticket) error {
	tk.UpdatedAt = time.Now()
	return s.tickets.Update(ctx, tk)
}

func (s *TicketService) List(ctx context.Context, tenantID int64, offset, limit int) ([]*Ticket, error) {
	return s.tickets.List(ctx, tenantID, offset, limit)
}

func (s *TicketService) ListByCallID(ctx context.Context, callID int64) ([]*Ticket, error) {
	return s.tickets.ListByCallID(ctx, callID)
}

func (s *TicketService) Assign(ctx context.Context, ticketID, agentID int64) (*Ticket, error) {
	tk, err := s.tickets.GetByID(ctx, ticketID)
	if err != nil || tk == nil {
		return nil, ErrTicketNotFound
	}

	tk.AssigneeID = &agentID
	tk.UpdatedAt = time.Now()
	if err := s.tickets.Update(ctx, tk); err != nil {
		return nil, err
	}
	return tk, nil
}

func (s *TicketService) Transition(ctx context.Context, ticketID int64, newStatus TicketStatus) (*Ticket, error) {
	tk, err := s.tickets.GetByID(ctx, ticketID)
	if err != nil || tk == nil {
		return nil, ErrTicketNotFound
	}

	allowed := validTransitions[tk.Status]
	valid := false
	for _, s := range allowed {
		if s == newStatus {
			valid = true
			break
		}
	}
	if !valid {
		return nil, ErrInvalidTransition
	}

	tk.Status = newStatus
	tk.UpdatedAt = time.Now()
	if newStatus == TicketStatusResolved {
		now := time.Now()
		tk.ResolvedAt = &now
	}

	if err := s.tickets.Update(ctx, tk); err != nil {
		return nil, err
	}
	return tk, nil
}

func (s *TicketService) AddComment(ctx context.Context, ticketID, authorID int64, content string) error {
	c := &TicketComment{
		ID:        snowflake.NextID(),
		TicketID:  ticketID,
		AuthorID:  authorID,
		Content:   content,
		CreatedAt: time.Now(),
	}
	return s.comments.Create(ctx, c)
}

func (s *TicketService) ListComments(ctx context.Context, ticketID int64) ([]*TicketComment, error) {
	return s.comments.ListByTicket(ctx, ticketID)
}
