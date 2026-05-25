package ivr

import (
	"context"

	"github.com/divord97/ccc/internal/domain/routing"
)

// CustomerLookupProvider looks up a customer by phone number in the CRM.
type CustomerLookupProvider interface {
	LookupByPhone(ctx context.Context, tenantID int64, phone string) (name, level string, err error)
}

// CustomerLookupHandler queries CRM during IVR to set caller variables
// (customer_name, customer_level) and route VIP/SVIP callers through a
// priority exit so they can receive elevated ACD priority.
type CustomerLookupHandler struct {
	CRM CustomerLookupProvider
}

func (h *CustomerLookupHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	phone := sess.Variables["caller_number"]
	if phone == "" || h.CRM == nil {
		sess.Variables["customer_level"] = "unknown"
		return "default", nil
	}

	tenantID := sess.TenantID
	name, level, err := h.CRM.LookupByPhone(ctx, tenantID, phone)
	if err != nil {
		sess.Variables["customer_level"] = "unknown"
		return "not_found", nil
	}

	sess.Variables["customer_name"] = name
	sess.Variables["customer_level"] = level

	switch level {
	case "svip":
		return "svip", nil
	case "vip":
		return "vip", nil
	default:
		return "default", nil
	}
}
