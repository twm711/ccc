// Package acd implements the Automatic Call Distribution dispatcher.
//
// Background: prior to this package, calls were routed to FreeSWITCH's
// mod_callcenter (`callcenter:{skill_group_id}@default`) but no
// callcenter.conf.xml was deployed, so calls entered the queue and were never
// distributed. This package replaces that integration with a server-side
// dispatcher that:
//
//  1. Accepts Enqueue requests when a call leaves IVR (skill group, priority).
//  2. Stores queued calls in a Redis sorted set per skill group.
//  3. Polls each active skill group on a ticker; for each head-of-queue call,
//     selects an idle agent according to the configured routing policy and
//     transitions the call to ringing via lifecycle.Service.
//
// The dispatcher runs as a single goroutine. For multi-instance deployments,
// callers can scope the loop to a subset of skill groups; the Redis state
// itself is shared so any instance can drain.
package acd

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/divord97/ccc/internal/application/lifecycle"
	"github.com/divord97/ccc/internal/domain/call"
	"github.com/divord97/ccc/internal/domain/identity"
	"github.com/divord97/ccc/pkg/metrics"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

const (
	queueKeyPrefix    = "acd:queue:"       // ZSET: score = priority-adjusted enqueue time
	activeSGKey       = "acd:active_sg"    // SET of skill_group_ids that have ever been used
	agentClaimPrefix  = "acd:agent_claim:" // SETNX agent_id during dispatch to avoid double-route
	roundRobinPrefix  = "acd:rr_cursor:"   // INCR cursor per skill group
	lastAgentPrefix   = "acd:last_agent:"  // key: {tenant}:{caller} -> agent_user_id (familiar policy)
	defaultPollPeriod = 500 * time.Millisecond
	agentClaimTTL     = 30 * time.Second
)

// LifecycleService is the subset of lifecycle.Service required by the dispatcher.
type LifecycleService interface {
	TransitionCallToRinging(ctx context.Context, callID, agentUserID int64) (*call.Call, error)
	EndCall(ctx context.Context, callID int64, reason call.HangupReason, hangupBy ...call.HangupBy) (*call.Call, error)
}

var _ LifecycleService = (*lifecycle.Service)(nil)

// PresenceRepo exposes the lookups the dispatcher needs.
type PresenceRepo interface {
	GetByAgentID(ctx context.Context, agentID int64) (*identity.AgentPresence, error)
	GetByAgentIDs(ctx context.Context, agentIDs []int64) ([]*identity.AgentPresence, error)
}

// MembersRepo lists agents in a skill group.
type MembersRepo interface {
	GetBySkillGroup(ctx context.Context, skillGroupID int64) ([]*identity.SkillGroupMember, error)
}

// SkillGroups resolves the routing policy and tenant for a skill group.
type SkillGroups interface {
	GetByID(ctx context.Context, id int64) (*identity.SkillGroup, error)
}

// CallLookup loads a queued call so the dispatcher can read its caller phone
// number (needed by the familiar-customer routing policy). Optional; when nil
// familiar falls back to longest-idle.
type CallLookup interface {
	GetByID(ctx context.Context, id int64) (*call.Call, error)
}

// AgentLookup resolves an agent record to read per-media capacity settings.
type AgentLookup interface {
	GetByID(ctx context.Context, id int64) (*identity.Agent, error)
}

// Service is the ACD dispatcher.
type Service struct {
	rdb        *redis.Client
	lifecycle  LifecycleService
	presence   PresenceRepo
	members    MembersRepo
	skillGroup SkillGroups
	calls      CallLookup
	agentLookup AgentLookup
	logger     zerolog.Logger
	pollPeriod time.Duration
	rngMu      sync.Mutex
	rng        *rand.Rand
}

// Config groups the dependencies for NewService.
type Config struct {
	Redis      *redis.Client
	Lifecycle  LifecycleService
	Presence   PresenceRepo
	Members    MembersRepo
	SkillGroup SkillGroups
	Calls      CallLookup
	Logger     zerolog.Logger
	PollPeriod time.Duration
}

// SetAgentLookup wires agent lookup for media capacity routing.
func (s *Service) SetAgentLookup(al AgentLookup) { s.agentLookup = al }

// NewService wires the ACD dispatcher. The returned service is inert until Run is called.
func NewService(cfg Config) *Service {
	pp := cfg.PollPeriod
	if pp <= 0 {
		pp = defaultPollPeriod
	}
	return &Service{
		rdb:        cfg.Redis,
		lifecycle:  cfg.Lifecycle,
		presence:   cfg.Presence,
		members:    cfg.Members,
		skillGroup: cfg.SkillGroup,
		calls:      cfg.Calls,
		logger:     cfg.Logger,
		pollPeriod: pp,
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Enqueue appends a call to the skill group queue. Higher priority values are
// served first; equal priorities are FIFO.
func (s *Service) Enqueue(ctx context.Context, callID, skillGroupID int64, priority int) error {
	if s.rdb == nil {
		return errors.New("acd: redis client not configured")
	}

	// Check max_queue_size before enqueuing; overflow to backup group when possible.
	if sg, err := s.skillGroup.GetByID(ctx, skillGroupID); err == nil && sg != nil && sg.MaxQueueSize > 0 {
		qLen, _ := s.rdb.ZCard(ctx, queueKey(skillGroupID)).Result()
		if qLen >= int64(sg.MaxQueueSize) {
			if sg.OverflowGroup != nil {
				s.logger.Info().Int64("call_id", callID).Int64("from_sg", skillGroupID).Int64("to_sg", *sg.OverflowGroup).Msg("acd: queue full, overflowing to backup group")
				return s.Enqueue(ctx, callID, *sg.OverflowGroup, priority)
			}
			s.logger.Warn().Int64("call_id", callID).Int64("sg", skillGroupID).Int("max", sg.MaxQueueSize).Msg("acd: queue full, rejecting")
			metrics.QueueRejected.Inc()
			return fmt.Errorf("acd: queue full for skill group %d (max %d)", skillGroupID, sg.MaxQueueSize)
		}
	}

	now := time.Now()
	score := scoreFor(priority, now)
	if err := s.rdb.ZAdd(ctx, queueKey(skillGroupID), redis.Z{Score: score, Member: memberFor(callID, now)}).Err(); err != nil {
		return fmt.Errorf("acd: enqueue zadd: %w", err)
	}
	if err := s.rdb.SAdd(ctx, activeSGKey, skillGroupID).Err(); err != nil {
		return fmt.Errorf("acd: register sg: %w", err)
	}
	metrics.QueueEnqueued.Inc()
	s.logger.Debug().Int64("call_id", callID).Int64("sg", skillGroupID).Int("priority", priority).Msg("acd: enqueued")
	return nil
}

// Run drives the dispatcher loop until ctx is canceled.
func (s *Service) Run(ctx context.Context) {
	if s.rdb == nil {
		s.logger.Warn().Msg("acd: redis not configured, dispatcher disabled")
		return
	}
	t := time.NewTicker(s.pollPeriod)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.tick(ctx)
		}
	}
}

func (s *Service) tick(ctx context.Context) {
	sgIDs, err := s.rdb.SMembers(ctx, activeSGKey).Result()
	if err != nil {
		s.logger.Warn().Err(err).Msg("acd: list skill groups")
		return
	}
	// Pre-load skill groups once per tick to avoid repeated DB lookups.
	sgCache := make(map[int64]*identity.SkillGroup, len(sgIDs))
	for _, raw := range sgIDs {
		sgID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			continue
		}
		sg, err := s.skillGroup.GetByID(ctx, sgID)
		if err != nil || sg == nil {
			continue
		}
		sgCache[sgID] = sg
	}
	for sgID, sg := range sgCache {
		s.expireQueued(ctx, sgID, sg)
		s.dispatchOne(ctx, sgID, sg)
	}
}

// expireQueued drains calls whose wait time has exceeded the skill group
// max_wait_sec setting, marking them abandoned via lifecycle.EndCall so that
// post-call hooks (recording, webhook, etc.) still run.
//
// Enqueue time is read from the member string (memberFor format), not the
// score: the score mixes priority + timestamp in a way that cannot be cleanly
// decoded for arbitrary priority magnitudes, so we keep wall-clock time
// authoritative by embedding it in the member.
func (s *Service) expireQueued(ctx context.Context, sgID int64, sg *identity.SkillGroup) {
	if sg.MaxWaitSec <= 0 {
		return
	}
	entries, err := s.rdb.ZRange(ctx, queueKey(sgID), 0, -1).Result()
	if err != nil {
		return
	}
	deadline := time.Now().Add(-time.Duration(sg.MaxWaitSec) * time.Second).UnixMilli()
	for _, member := range entries {
		callID, enqueuedAt, ok := parseMember(member)
		if !ok {
			// Malformed entry; drop it so it doesn't wedge the queue.
			_ = s.rdb.ZRem(ctx, queueKey(sgID), member).Err()
			continue
		}
		if enqueuedAt == 0 {
			// Legacy member with no timestamp; age unknown — leave for dispatcher.
			continue
		}
		if enqueuedAt > deadline {
			continue
		}
		removed, err := s.rdb.ZRem(ctx, queueKey(sgID), member).Result()
		if err != nil || removed == 0 {
			continue
		}
		if _, err := s.lifecycle.EndCall(ctx, callID, call.HangupQueueTimeout, call.HangupBySystem); err != nil {
			s.logger.Warn().Err(err).Int64("call_id", callID).Int64("sg", sgID).Msg("acd: queue timeout end call")
			continue
		}
		s.logger.Info().Int64("call_id", callID).Int64("sg", sgID).Int("max_wait_sec", sg.MaxWaitSec).Msg("acd: call abandoned (queue timeout)")
	}
}

// dispatchOne attempts to assign the head-of-queue call for a skill group to an
// idle agent. At most one assignment per tick per skill group to keep the loop
// fair across groups.
func (s *Service) dispatchOne(ctx context.Context, sgID int64, sg *identity.SkillGroup) {
	head, err := s.rdb.ZRangeWithScores(ctx, queueKey(sgID), 0, 0).Result()
	if err != nil || len(head) == 0 {
		return
	}
	member, _ := head[0].Member.(string)
	callID, _, ok := parseMember(member)
	if !ok {
		_ = s.rdb.ZRem(ctx, queueKey(sgID), head[0].Member).Err()
		return
	}

	agentID, err := s.pickAgent(ctx, sg, callID)
	if err != nil || agentID == 0 {
		return
	}

	if !s.tryClaim(ctx, agentID) {
		return
	}

	removed, err := s.rdb.ZRem(ctx, queueKey(sgID), member).Result()
	if err != nil || removed == 0 {
		s.releaseClaim(ctx, agentID)
		return
	}

	if _, err := s.lifecycle.TransitionCallToRinging(ctx, callID, agentID); err != nil {
		s.logger.Warn().Err(err).Int64("call_id", callID).Int64("agent_id", agentID).Msg("acd: transition to ringing failed")
		s.releaseClaim(ctx, agentID)
		// Requeue with original score (preserves priority window + original
		// enqueue timestamp) so a transient failure doesn't demote the call.
		// member carries the enqueue timestamp suffix so expireQueued can
		// still tell how long the call has been waiting overall.
		_ = s.rdb.ZAdd(ctx, queueKey(sgID), redis.Z{Score: head[0].Score, Member: member}).Err()
		return
	}
	if _, enqueuedAt, ok := parseMember(member); ok && enqueuedAt > 0 {
		metrics.ACDDispatchLatency.Observe(float64(time.Now().UnixMilli()-enqueuedAt) / 1000.0)
	}
	s.logger.Info().Int64("call_id", callID).Int64("agent_id", agentID).Int64("sg", sgID).Msg("acd: routed call to agent")
}

func (s *Service) pickAgent(ctx context.Context, sg *identity.SkillGroup, callID int64) (int64, error) {
	members, err := s.members.GetBySkillGroup(ctx, sg.ID)
	if err != nil {
		return 0, err
	}
	type idleAgent struct {
		ID       int64
		LastIdle time.Time
	}
	// Batch-fetch presence to avoid N MySQL round-trips per dispatch.
	agentIDs := make([]int64, 0, len(members))
	for _, m := range members {
		agentIDs = append(agentIDs, m.AgentID)
	}
	presences, err := s.presence.GetByAgentIDs(ctx, agentIDs)
	if err != nil {
		return 0, err
	}
	byAgent := make(map[int64]*identity.AgentPresence, len(presences))
	for _, p := range presences {
		byAgent[p.AgentID] = p
	}
	var candidates []idleAgent
	for _, m := range members {
		p := byAgent[m.AgentID]
		if p == nil || p.Status != identity.PresenceIdle {
			continue
		}
		candidates = append(candidates, idleAgent{ID: m.AgentID, LastIdle: p.LastStatusAt})
	}
	if len(candidates) == 0 {
		return 0, nil
	}

	switch sg.RoutingPolicy {
	case identity.RoutingPolicyRandom:
		s.rngMu.Lock()
		idx := s.rng.Intn(len(candidates))
		s.rngMu.Unlock()
		return candidates[idx].ID, nil
	case identity.RoutingPolicyRoundRobin:
		idx, err := s.rdb.Incr(ctx, roundRobinPrefix+strconv.FormatInt(sg.ID, 10)).Result()
		if err != nil {
			return candidates[0].ID, nil
		}
		return candidates[int((idx-1)%int64(len(candidates)))].ID, nil
	case identity.RoutingPolicyFamiliar:
		if preferred := s.lookupFamiliarAgent(ctx, callID); preferred > 0 {
			for _, c := range candidates {
				if c.ID == preferred {
					return preferred, nil
				}
			}
		}
		fallthrough
	default:
		// longest-idle (least_recent / skill_weight all fall back to longest-idle;
		// familiar falls through here when there is no recent agent or that agent is not idle).
		best := candidates[0]
		for _, c := range candidates[1:] {
			if c.LastIdle.Before(best.LastIdle) {
				best = c
			}
		}
		return best.ID, nil
	}
}

// lookupFamiliarAgent returns the agent_user_id who most recently served the
// caller of the given call, or 0 if no record exists.
func (s *Service) lookupFamiliarAgent(ctx context.Context, callID int64) int64 {
	if s.calls == nil {
		return 0
	}
	c, err := s.calls.GetByID(ctx, callID)
	if err != nil || c == nil || c.Caller == "" {
		return 0
	}
	val, err := s.rdb.Get(ctx, familiarKey(c.TenantID, c.Caller)).Result()
	if err != nil || val == "" {
		return 0
	}
	agentID, _ := strconv.ParseInt(val, 10, 64)
	return agentID
}

// RememberAgent records that agentID served the given caller. ttlDays caps how
// long the affinity is kept; pass 0 to disable (RememberAgent becomes a no-op).
// Called by lifecycle.Service.EndCall.
func (s *Service) RememberAgent(ctx context.Context, tenantID int64, caller string, agentUserID int64, ttlDays int) {
	if s.rdb == nil || caller == "" || agentUserID == 0 || ttlDays <= 0 {
		return
	}
	ttl := time.Duration(ttlDays) * 24 * time.Hour
	_ = s.rdb.Set(ctx, familiarKey(tenantID, caller), strconv.FormatInt(agentUserID, 10), ttl).Err()
}

func familiarKey(tenantID int64, caller string) string {
	return fmt.Sprintf("%s%d:%s", lastAgentPrefix, tenantID, caller)
}

// GetMediaCapacity returns an agent's capacity across all media types.
// activeVoice/activeChat/activeEmail are read from Redis counters if available.
func (s *Service) GetMediaCapacity(ctx context.Context, agentID int64) ([]identity.MediaCapacity, error) {
	if s.agentLookup == nil {
		return nil, errors.New("acd: agent lookup not configured")
	}
	agent, err := s.agentLookup.GetByID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	voiceMax := agent.MaxConcurrent
	if voiceMax == 0 {
		voiceMax = 1
	}
	chatMax := agent.MaxChatSlots
	if chatMax == 0 {
		chatMax = 5
	}
	emailMax := agent.MaxEmailSlots
	if emailMax == 0 {
		emailMax = 10
	}

	activeVoice := s.readActiveCount(ctx, agentID, "voice")
	activeChat := s.readActiveCount(ctx, agentID, "chat")
	activeEmail := s.readActiveCount(ctx, agentID, "email")

	return []identity.MediaCapacity{
		{Media: identity.MediaTypeVoice, MaxSlots: voiceMax, ActiveSlots: activeVoice},
		{Media: identity.MediaTypeChat, MaxSlots: chatMax, ActiveSlots: activeChat},
		{Media: identity.MediaTypeEmail, MaxSlots: emailMax, ActiveSlots: activeEmail},
	}, nil
}

// AgentWorkloadSummary aggregates an agent's current workload across all channels.
type AgentWorkloadSummary struct {
	AgentID    int64                    `json:"agent_id"`
	Capacities []identity.MediaCapacity `json:"capacities"`
	TotalActive int                    `json:"total_active"`
	TotalMax    int                    `json:"total_max"`
	Utilization float64                `json:"utilization"` // 0-100
}

// GetWorkloadSummary returns an agent's multi-channel workload snapshot.
func (s *Service) GetWorkloadSummary(ctx context.Context, agentID int64) (*AgentWorkloadSummary, error) {
	caps, err := s.GetMediaCapacity(ctx, agentID)
	if err != nil {
		return nil, err
	}
	var totalActive, totalMax int
	for _, c := range caps {
		totalActive += c.ActiveSlots
		totalMax += c.MaxSlots
	}
	var util float64
	if totalMax > 0 {
		util = float64(totalActive) / float64(totalMax) * 100
	}
	return &AgentWorkloadSummary{
		AgentID:     agentID,
		Capacities:  caps,
		TotalActive: totalActive,
		TotalMax:    totalMax,
		Utilization: util,
	}, nil
}

func (s *Service) readActiveCount(ctx context.Context, agentID int64, media string) int {
	key := fmt.Sprintf("acd:active:%d:%s", agentID, media)
	val, err := s.rdb.Get(ctx, key).Int()
	if err != nil {
		return 0
	}
	return val
}

func (s *Service) tryClaim(ctx context.Context, agentID int64) bool {
	ok, err := s.rdb.SetNX(ctx, agentClaimPrefix+strconv.FormatInt(agentID, 10), 1, agentClaimTTL).Result()
	return err == nil && ok
}

func (s *Service) releaseClaim(ctx context.Context, agentID int64) {
	_ = s.rdb.Del(ctx, agentClaimPrefix+strconv.FormatInt(agentID, 10)).Err()
}

func queueKey(sgID int64) string { return queueKeyPrefix + strconv.FormatInt(sgID, 10) }

// scoreFor encodes priority + timestamp into a single ZSet score so higher
// priority always sorts before older entries with lower priority.
func scoreFor(priority int, ts time.Time) float64 {
	// Priority window: -1e10 per priority point, then +ms since epoch.
	// Note: this is one-way; do NOT try to decode priority/timestamp from the
	// score — the window (1e10) is smaller than current ms-since-epoch
	// (~1.78e12), so dividing the score by the window produces nonsense.
	// Timestamps are stored authoritatively in the member string (memberFor).
	return float64(-priority)*1e10 + float64(ts.UnixMilli())
}

// memberFor encodes the queue member as "<callID>:<enqueueMs>" so the
// dispatcher and expireQueued can recover the enqueue time without trying to
// decode it from the score.
func memberFor(callID int64, enqueuedAt time.Time) string {
	return strconv.FormatInt(callID, 10) + ":" + strconv.FormatInt(enqueuedAt.UnixMilli(), 10)
}

// parseMember parses a queue member produced by memberFor.
//
//   - ok=true with enqueueMs>0: new-format entry, age known.
//   - ok=true with enqueueMs==0: legacy entry (bare call ID, no ':' suffix);
//     the call ID is usable but the age is unknown, so expireQueued should
//     skip it and let the dispatcher drain it normally.
//   - ok=false: malformed (cannot recover a call ID); the caller should ZREM.
func parseMember(member string) (callID int64, enqueueMs int64, ok bool) {
	i := strings.IndexByte(member, ':')
	if i < 0 {
		id, err := strconv.ParseInt(member, 10, 64)
		if err != nil {
			return 0, 0, false
		}
		return id, 0, true
	}
	id, err := strconv.ParseInt(member[:i], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	ms, err := strconv.ParseInt(member[i+1:], 10, 64)
	if err != nil {
		return id, 0, false
	}
	return id, ms, true
}
