package ai

import "time"

// ConversationState represents the current state in a multi-turn dialog.
type ConversationState string

const (
	StateGreeting     ConversationState = "greeting"
	StateIdentify     ConversationState = "identify"
	StateInquiry      ConversationState = "inquiry"
	StateConfirm      ConversationState = "confirm"
	StateTransfer     ConversationState = "transfer"
	StateCompleted    ConversationState = "completed"
)

// ConversationSession tracks multi-turn dialog state for a digital employee.
type ConversationSession struct {
	SessionID         string            `json:"session_id"`
	DigitalEmployeeID int64             `json:"digital_employee_id"`
	SceneID           int64             `json:"scene_id"`
	TenantID          int64             `json:"tenant_id"`
	CallerID          string            `json:"caller_id"`
	State             ConversationState `json:"state"`
	TurnCount         int               `json:"turn_count"`
	Slots             map[string]string `json:"slots"`
	History           []DialogTurn      `json:"history"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// DialogTurn is a single exchange in the conversation.
type DialogTurn struct {
	Role    string `json:"role"` // "user" or "bot"
	Content string `json:"content"`
	Intent  string `json:"intent,omitempty"`
}

// Transition represents a valid state transition in the dialog FSM.
type Transition struct {
	From      ConversationState
	Trigger   string // intent name or special trigger
	To        ConversationState
	Action    string // action to perform on transition (e.g. "ask_name", "lookup_order")
}

// DialogFSM is a finite state machine for multi-turn conversations.
type DialogFSM struct {
	Transitions []Transition
	Initial     ConversationState
}

// DefaultDialogFSM returns a standard multi-turn conversation flow.
func DefaultDialogFSM() *DialogFSM {
	return &DialogFSM{
		Initial: StateGreeting,
		Transitions: []Transition{
			{From: StateGreeting, Trigger: "*", To: StateIdentify, Action: "ask_identity"},
			{From: StateIdentify, Trigger: "identified", To: StateInquiry, Action: "ask_inquiry"},
			{From: StateIdentify, Trigger: "skip", To: StateInquiry, Action: "ask_inquiry"},
			{From: StateInquiry, Trigger: "matched", To: StateConfirm, Action: "confirm_intent"},
			{From: StateInquiry, Trigger: "transfer", To: StateTransfer, Action: "transfer_agent"},
			{From: StateInquiry, Trigger: "unmatched", To: StateInquiry, Action: "ask_again"},
			{From: StateConfirm, Trigger: "yes", To: StateCompleted, Action: "resolve"},
			{From: StateConfirm, Trigger: "no", To: StateInquiry, Action: "ask_inquiry"},
			{From: StateConfirm, Trigger: "transfer", To: StateTransfer, Action: "transfer_agent"},
		},
	}
}

// Next finds the next state for a given current state and trigger.
// Returns the transition if found, or nil if no matching transition exists.
func (fsm *DialogFSM) Next(current ConversationState, trigger string) *Transition {
	for i := range fsm.Transitions {
		t := &fsm.Transitions[i]
		if t.From == current && (t.Trigger == trigger || t.Trigger == "*") {
			return t
		}
	}
	return nil
}

// NewConversationSession creates a new session at the FSM's initial state.
func NewConversationSession(sessionID string, deID, sceneID, tenantID int64, callerID string) *ConversationSession {
	now := time.Now()
	return &ConversationSession{
		SessionID:         sessionID,
		DigitalEmployeeID: deID,
		SceneID:           sceneID,
		TenantID:          tenantID,
		CallerID:          callerID,
		State:             StateGreeting,
		Slots:             make(map[string]string),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// AddTurn appends a dialog turn and increments the counter.
func (s *ConversationSession) AddTurn(role, content, intent string) {
	s.History = append(s.History, DialogTurn{Role: role, Content: content, Intent: intent})
	s.TurnCount++
	s.UpdatedAt = time.Now()
}

// Advance moves the session to the next state using the FSM.
// Returns the action to perform, or "" if no valid transition exists.
func (s *ConversationSession) Advance(fsm *DialogFSM, trigger string) string {
	t := fsm.Next(s.State, trigger)
	if t == nil {
		return ""
	}
	s.State = t.To
	s.UpdatedAt = time.Now()
	return t.Action
}

// IsTerminal returns true if the session has reached a terminal state.
func (s *ConversationSession) IsTerminal() bool {
	return s.State == StateCompleted || s.State == StateTransfer
}
