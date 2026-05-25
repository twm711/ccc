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
)
