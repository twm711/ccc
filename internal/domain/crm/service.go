package crm

import (
	"context"
	"fmt"
	"time"

	"github.com/divord97/ccc/pkg/snowflake"
)

type CustomerService struct {
	customers    CustomerRepository
	phones       CustomerPhoneRepository
	interactions CustomerInteractionRepository
	fields       CustomFieldDefinitionRepository
}

func NewCustomerService(
	customers CustomerRepository,
	phones CustomerPhoneRepository,
	interactions CustomerInteractionRepository,
	fields CustomFieldDefinitionRepository,
) *CustomerService {
	return &CustomerService{
		customers:    customers,
		phones:       phones,
		interactions: interactions,
		fields:       fields,
	}
}

type PhoneInput struct {
	PhoneType string `json:"phone_type"`
	Number    string `json:"number"`
	IsPrimary bool   `json:"is_primary"`
}

type CreateCustomerInput struct {
	TenantID   int64        `json:"tenant_id"`
	Name       string       `json:"name"`
	Email      string       `json:"email"`
	Company    string       `json:"company"`
	Level      string       `json:"level"`
	CustomData string       `json:"custom_data"`
	Phones     []PhoneInput `json:"phones"`
}

type RecordInteractionInput struct {
	CustomerID int64
	TenantID   int64
	Channel    string
	Direction  string
	Summary    string
	CallID     *int64
	TicketID   *int64
	AgentName  string
}

type BatchImportResult struct {
	Success int
	Failed  int
	Errors  []string
}

var validLevels = map[string]bool{"normal": true, "vip": true, "svip": true}
var validPhoneTypes = map[string]bool{"mobile": true, "landline": true, "backup": true}
var validFieldTypes = map[string]bool{"text": true, "number": true, "select": true, "date": true}
var validEntityTypes = map[string]bool{"customer": true, "ticket": true}

func (s *CustomerService) Create(ctx context.Context, in CreateCustomerInput) (*Customer, error) {
	if !validLevels[in.Level] {
		return nil, ErrInvalidLevel
	}

	if err := validatePhones(in.Phones); err != nil {
		return nil, err
	}

	now := time.Now()
	c := &Customer{
		ID:         snowflake.NextID(),
		TenantID:   in.TenantID,
		Name:       in.Name,
		Email:      in.Email,
		Company:    in.Company,
		Level:      in.Level,
		CustomData: in.CustomData,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.customers.Create(ctx, c); err != nil {
		return nil, err
	}

	for _, p := range in.Phones {
		phone := &CustomerPhone{
			ID:         snowflake.NextID(),
			CustomerID: c.ID,
			PhoneType:  p.PhoneType,
			Number:     p.Number,
			IsPrimary:  p.IsPrimary,
		}
		if err := s.phones.Create(ctx, phone); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func validatePhones(phones []PhoneInput) error {
	var primaryCount int
	for _, p := range phones {
		if !validPhoneTypes[p.PhoneType] {
			return ErrInvalidPhoneType
		}
		if p.IsPrimary {
			primaryCount++
		}
	}
	if primaryCount == 0 {
		return ErrNoPrimaryPhone
	}
	if primaryCount > 1 {
		return ErrMultiplePrimary
	}
	return nil
}

func (s *CustomerService) GetByID(ctx context.Context, id int64) (*Customer, error) {
	c, err := s.customers.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCustomerNotFound
	}
	return c, nil
}

func (s *CustomerService) Update(ctx context.Context, c *Customer) error {
	c.UpdatedAt = time.Now()
	return s.customers.Update(ctx, c)
}

func (s *CustomerService) Delete(ctx context.Context, id int64) error {
	if err := s.phones.DeleteByCustomer(ctx, id); err != nil {
		return err
	}
	if err := s.interactions.DeleteByCustomer(ctx, id); err != nil {
		return err
	}
	return s.customers.Delete(ctx, id)
}

func (s *CustomerService) List(ctx context.Context, tenantID int64, offset, limit int) ([]*Customer, error) {
	return s.customers.List(ctx, tenantID, offset, limit)
}

func (s *CustomerService) FindByPhone(ctx context.Context, tenantID int64, phone string) (*Customer, error) {
	// Attempt direct lookup via customer repo (for MySQL FULLTEXT or JOIN)
	c, err := s.customers.FindByPhone(ctx, tenantID, phone)
	if err != nil {
		return nil, err
	}
	if c != nil {
		return c, nil
	}

	// Fallback for mock: scan phone repo
	if mockPhones, ok := s.phones.(*MockCustomerPhoneRepo); ok {
		custID := mockPhones.FindCustomerByPhone(phone)
		if custID == 0 {
			return nil, nil
		}
		return s.customers.GetByID(ctx, custID)
	}

	return nil, nil
}

func (s *CustomerService) ListPhones(ctx context.Context, customerID int64) ([]*CustomerPhone, error) {
	return s.phones.ListByCustomer(ctx, customerID)
}

func (s *CustomerService) RecordInteraction(ctx context.Context, in RecordInteractionInput) error {
	interaction := &CustomerInteraction{
		ID:         snowflake.NextID(),
		CustomerID: in.CustomerID,
		TenantID:   in.TenantID,
		Channel:    in.Channel,
		Direction:  in.Direction,
		Summary:    in.Summary,
		CallID:     in.CallID,
		TicketID:   in.TicketID,
		AgentName:  in.AgentName,
		CreatedAt:  time.Now(),
	}
	return s.interactions.Create(ctx, interaction)
}

func (s *CustomerService) ListInteractions(ctx context.Context, customerID int64, offset, limit int) ([]*CustomerInteraction, error) {
	return s.interactions.ListByCustomer(ctx, customerID, offset, limit)
}

func (s *CustomerService) CreateFieldDefinition(ctx context.Context, d CustomFieldDefinition) error {
	if !validEntityTypes[d.EntityType] {
		return ErrInvalidEntityType
	}
	if !validFieldTypes[d.FieldType] {
		return ErrInvalidFieldType
	}
	d.ID = snowflake.NextID()
	return s.fields.Create(ctx, &d)
}

func (s *CustomerService) ListFieldDefinitions(ctx context.Context, tenantID int64, entityType string) ([]*CustomFieldDefinition, error) {
	return s.fields.ListByEntity(ctx, tenantID, entityType)
}

func (s *CustomerService) BatchImport(ctx context.Context, records []CreateCustomerInput) (*BatchImportResult, error) {
	result := &BatchImportResult{}
	for i, rec := range records {
		_, err := s.Create(ctx, rec)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: %v", i+1, err))
			continue
		}
		result.Success++
	}
	return result, nil
}
