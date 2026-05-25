package ivr

import (
	"context"
	"strconv"

	"github.com/divord97/ccc/internal/domain/routing"
)

// QueueInspector returns the current queue depth for a skill group.
type QueueInspector interface {
	QueueLen(ctx context.Context, skillGroupID int64) (int, error)
}

// QueuePositionHandler announces the caller's estimated queue position and
// wait time before transferring to an agent. It sets IVR variables
// "queue_position" and "est_wait_sec" so downstream TTS nodes can voice them.
type QueuePositionHandler struct {
	Queue           QueueInspector
	AvgHandleTimeSec int // fallback AHT for wait estimation (default 180)
}

type queuePositionConfig struct {
	SkillGroupID string `json:"skill_group_id"`
}

func (h *QueuePositionHandler) Handle(ctx context.Context, sess *Session, node routing.FlowNode) (string, error) {
	var cfg queuePositionConfig
	if err := parseConfig(node.Config, &cfg); err != nil {
		return "error", nil
	}

	sgID, _ := strconv.ParseInt(cfg.SkillGroupID, 10, 64)
	if h.Queue == nil || sgID == 0 {
		sess.Variables["queue_position"] = "0"
		sess.Variables["est_wait_sec"] = "0"
		return "default", nil
	}

	depth, err := h.Queue.QueueLen(ctx, sgID)
	if err != nil {
		sess.Variables["queue_position"] = "0"
		sess.Variables["est_wait_sec"] = "0"
		return "error", nil
	}

	aht := h.AvgHandleTimeSec
	if aht <= 0 {
		aht = 180
	}
	estWait := depth * aht

	sess.Variables["queue_position"] = strconv.Itoa(depth)
	sess.Variables["est_wait_sec"] = strconv.Itoa(estWait)

	if depth == 0 {
		return "empty", nil
	}
	return "default", nil
}
