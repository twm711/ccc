package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CallsCreated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccc_calls_created_total",
			Help: "Total calls created by direction",
		},
		[]string{"direction"},
	)

	CallsEnded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccc_calls_ended_total",
			Help: "Total calls ended by hangup_by",
		},
		[]string{"hangup_by"},
	)

	ActiveCallsGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ccc_active_calls_total",
			Help: "Number of currently active calls",
		},
	)

	AgentStateTransitions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccc_agent_state_transitions_total",
			Help: "Agent state transitions",
		},
		[]string{"from", "to"},
	)

	QueueEnqueued = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ccc_queue_enqueued_total",
			Help: "Total calls enqueued to ACD",
		},
	)

	QueueRejected = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ccc_queue_rejected_total",
			Help: "Total calls rejected (queue full)",
		},
	)

	ConcurrencyRejected = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ccc_concurrency_rejected_total",
			Help: "Calls rejected due to tenant concurrency limit",
		},
	)

	// SLO metrics (P4-3)
	CallAnswerLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ccc_call_answer_latency_seconds",
			Help:    "Time from call creation to agent answer",
			Buckets: []float64{5, 10, 20, 30, 60, 120, 300},
		},
	)

	CallsAbandoned = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ccc_calls_abandoned_total",
			Help: "Total calls abandoned while queued",
		},
	)

	SLAMet = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ccc_sla_met_total",
			Help: "Calls answered within SLA threshold (20s)",
		},
	)

	SLAMissed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ccc_sla_missed_total",
			Help: "Calls answered after SLA threshold",
		},
	)

	// Capacity / saturation (P4-5)
	TenantActiveCalls = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ccc_tenant_active_calls",
			Help: "Active calls per tenant",
		},
		[]string{"tenant_id"},
	)

	TenantQueueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ccc_tenant_queue_depth",
			Help: "Queue depth per tenant",
		},
		[]string{"tenant_id"},
	)

	WSActiveConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ccc_ws_active_connections",
			Help: "Active WebSocket connections by hub type",
		},
		[]string{"hub"},
	)
)
