package bizlog

import (
	"github.com/rs/zerolog"
)

// Event emits a structured business event log entry. All fields are indexed
// for log aggregation (ELK, Loki) and enable querying by tenant, entity, or
// event type without parsing free-text messages.
func Event(logger zerolog.Logger, tenantID int64, eventType, entity string, entityID int64) *zerolog.Event {
	return logger.Info().
		Str("biz_event", eventType).
		Int64("tenant_id", tenantID).
		Str("entity", entity).
		Int64("entity_id", entityID)
}

// AgentEvent logs an agent-related business event.
func AgentEvent(logger zerolog.Logger, tenantID, agentID int64, eventType string) *zerolog.Event {
	return Event(logger, tenantID, eventType, "agent", agentID)
}

// CallEvent logs a call-related business event.
func CallEvent(logger zerolog.Logger, tenantID, callID int64, eventType string) *zerolog.Event {
	return Event(logger, tenantID, eventType, "call", callID)
}

// IMEvent logs an IM session business event.
func IMEvent(logger zerolog.Logger, tenantID, sessionID int64, eventType string) *zerolog.Event {
	return Event(logger, tenantID, eventType, "im_session", sessionID)
}

// CampaignEvent logs a campaign business event.
func CampaignEvent(logger zerolog.Logger, tenantID, campaignID int64, eventType string) *zerolog.Event {
	return Event(logger, tenantID, eventType, "campaign", campaignID)
}

// TicketEvent logs a ticket business event.
func TicketEvent(logger zerolog.Logger, tenantID, ticketID int64, eventType string) *zerolog.Event {
	return Event(logger, tenantID, eventType, "ticket", ticketID)
}
