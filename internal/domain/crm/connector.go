package crm

import "context"

// CRMConnector defines the standard interface for external CRM integrations.
type CRMConnector interface {
	Name() string
	// SyncContact pushes a customer to the external CRM, returning the external ID.
	SyncContact(ctx context.Context, tenantID int64, c *Customer) (externalID string, err error)
	// FetchContact pulls a contact from the external CRM by external ID.
	FetchContact(ctx context.Context, tenantID int64, externalID string) (*Customer, error)
	// PushInteraction records an interaction in the external CRM.
	PushInteraction(ctx context.Context, tenantID int64, externalID string, ix *CustomerInteraction) error
}

// ConnectorRegistry manages available CRM connectors per tenant.
type ConnectorRegistry struct {
	connectors map[string]CRMConnector
}

// NewConnectorRegistry creates an empty registry.
func NewConnectorRegistry() *ConnectorRegistry {
	return &ConnectorRegistry{connectors: make(map[string]CRMConnector)}
}

// Register adds a connector.
func (r *ConnectorRegistry) Register(c CRMConnector) { r.connectors[c.Name()] = c }

// Get returns a connector by name.
func (r *ConnectorRegistry) Get(name string) (CRMConnector, bool) {
	c, ok := r.connectors[name]
	return c, ok
}

// List returns all registered connector names.
func (r *ConnectorRegistry) List() []string {
	names := make([]string, 0, len(r.connectors))
	for n := range r.connectors {
		names = append(names, n)
	}
	return names
}
