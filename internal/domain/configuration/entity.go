package configuration

import "time"

type BreakReason struct {
	ID        int64     `db:"id" json:"id"`
	TenantID  int64     `db:"tenant_id" json:"tenant_id"`
	Code      string    `db:"code" json:"code"`
	Name      string    `db:"name" json:"name"`
	IsSystem  bool      `db:"is_system" json:"is_system"`
	SortOrder int       `db:"sort_order" json:"sort_order"`
	Enabled   bool      `db:"enabled" json:"enabled"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type DispositionCode struct {
	ID        int64     `db:"id" json:"id"`
	TenantID  int64     `db:"tenant_id" json:"tenant_id"`
	Code      string    `db:"code" json:"code"`
	Name      string    `db:"name" json:"name"`
	Category  string    `db:"category" json:"category"`
	SortOrder int       `db:"sort_order" json:"sort_order"`
	Enabled   bool      `db:"enabled" json:"enabled"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type CustomFieldTarget string

const (
	CustomFieldTargetCustomer CustomFieldTarget = "customer"
	CustomFieldTargetTicket   CustomFieldTarget = "ticket"
	CustomFieldTargetCall     CustomFieldTarget = "call"
)

type CustomFieldType string

const (
	CustomFieldTypeText     CustomFieldType = "text"
	CustomFieldTypeNumber   CustomFieldType = "number"
	CustomFieldTypeDate     CustomFieldType = "date"
	CustomFieldTypeSelect   CustomFieldType = "select"
	CustomFieldTypeCheckbox CustomFieldType = "checkbox"
)

type CustomFieldDefinition struct {
	ID           int64             `db:"id" json:"id"`
	TenantID     int64             `db:"tenant_id" json:"tenant_id"`
	Target       CustomFieldTarget `db:"target" json:"target"`
	FieldKey     string            `db:"field_key" json:"field_key"`
	FieldLabel   string            `db:"field_label" json:"field_label"`
	FieldType    CustomFieldType   `db:"field_type" json:"field_type"`
	Options      string            `db:"options" json:"options,omitempty"`
	Required     bool              `db:"required" json:"required"`
	SortOrder    int               `db:"sort_order" json:"sort_order"`
	CreatedAt    time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time         `db:"updated_at" json:"updated_at"`
}

type CallTag struct {
	ID        int64     `db:"id" json:"id"`
	TenantID  int64     `db:"tenant_id" json:"tenant_id"`
	Name      string    `db:"name" json:"name"`
	Color     string    `db:"color" json:"color"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
