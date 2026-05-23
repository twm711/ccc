package im

import (
	"context"
	"testing"

	"github.com/divord97/ccc/pkg/snowflake"
)

func init() {
	_ = snowflake.Init(1)
}

func newTestService(maxSlots int) *IMService {
	return NewIMService(
		NewMockIMChannelRepo(),
		NewMockIMSessionRepo(),
		NewMockIMMessageRepo(),
		maxSlots,
	)
}

func createTestChannel(t *testing.T, svc *IMService, tenantID int64) *IMChannel {
	t.Helper()
	ch, err := svc.CreateChannel(context.Background(), tenantID, ChannelTypeWebWidget, "Test Widget", nil)
	if err != nil {
		t.Fatalf("createTestChannel: %v", err)
	}
	return ch
}

func TestIMService_CreateSession_RouteToSkillGroup(t *testing.T) {
	svc := newTestService(5)
	ctx := context.Background()

	sgID := int64(100)
	ch, err := svc.CreateChannel(ctx, 1, ChannelTypeWebWidget, "Support Widget", &sgID)
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	sess, err := svc.CreateSession(ctx, CreateSessionInput{
		TenantID:  1,
		ChannelID: ch.ID,
		VisitorID: "visitor-001",
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if sess.Status != SessionStatusWaiting {
		t.Errorf("expected status waiting, got %s", sess.Status)
	}
	if sess.SkillGroupID == nil || *sess.SkillGroupID != sgID {
		t.Errorf("expected skill_group_id %d, got %v", sgID, sess.SkillGroupID)
	}

	// disabled channel should reject
	ch.Status = ChannelStatusDisabled
	_ = svc.UpdateChannel(ctx, ch)
	_, err = svc.CreateSession(ctx, CreateSessionInput{TenantID: 1, ChannelID: ch.ID, VisitorID: "v2"})
	if err != ErrChannelDisabled {
		t.Errorf("expected ErrChannelDisabled, got %v", err)
	}
}

func TestIMService_AssignAgent_LongestIdle(t *testing.T) {
	svc := newTestService(5)
	ctx := context.Background()

	ch := createTestChannel(t, svc, 1)
	sess, _ := svc.CreateSession(ctx, CreateSessionInput{TenantID: 1, ChannelID: ch.ID, VisitorID: "v1"})

	agentID := int64(200)
	if err := svc.AssignAgent(ctx, sess.ID, agentID); err != nil {
		t.Fatalf("AssignAgent: %v", err)
	}

	updated, _ := svc.GetSession(ctx, sess.ID)
	if updated.Status != SessionStatusActive {
		t.Errorf("expected active, got %s", updated.Status)
	}
	if updated.AgentUserID == nil || *updated.AgentUserID != agentID {
		t.Errorf("expected agent %d, got %v", agentID, updated.AgentUserID)
	}

	// cannot assign already active session
	if err := svc.AssignAgent(ctx, sess.ID, 300); err != ErrSessionNotWaiting {
		t.Errorf("expected ErrSessionNotWaiting, got %v", err)
	}
}

func TestIMService_TransferSession(t *testing.T) {
	svc := newTestService(5)
	ctx := context.Background()

	ch := createTestChannel(t, svc, 1)
	sess, _ := svc.CreateSession(ctx, CreateSessionInput{TenantID: 1, ChannelID: ch.ID, VisitorID: "v1"})
	_ = svc.AssignAgent(ctx, sess.ID, 200)

	if err := svc.TransferSession(ctx, sess.ID, 300); err != nil {
		t.Fatalf("TransferSession: %v", err)
	}
	updated, _ := svc.GetSession(ctx, sess.ID)
	if *updated.AgentUserID != 300 {
		t.Errorf("expected agent 300, got %d", *updated.AgentUserID)
	}
}

func TestIMService_CloseSession(t *testing.T) {
	svc := newTestService(5)
	ctx := context.Background()

	ch := createTestChannel(t, svc, 1)
	sess, _ := svc.CreateSession(ctx, CreateSessionInput{TenantID: 1, ChannelID: ch.ID, VisitorID: "v1"})
	_ = svc.AssignAgent(ctx, sess.ID, 200)

	if err := svc.CloseSession(ctx, sess.ID); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	updated, _ := svc.GetSession(ctx, sess.ID)
	if updated.Status != SessionStatusClosed {
		t.Errorf("expected closed, got %s", updated.Status)
	}
	if updated.EndAt == nil {
		t.Error("expected end_at to be set")
	}

	// double close
	if err := svc.CloseSession(ctx, sess.ID); err != ErrSessionClosed {
		t.Errorf("expected ErrSessionClosed, got %v", err)
	}
}

func TestIMService_SendMessage_Text(t *testing.T) {
	svc := newTestService(5)
	ctx := context.Background()

	ch := createTestChannel(t, svc, 1)
	sess, _ := svc.CreateSession(ctx, CreateSessionInput{TenantID: 1, ChannelID: ch.ID, VisitorID: "v1"})
	_ = svc.AssignAgent(ctx, sess.ID, 200)

	msg, err := svc.SendMessage(ctx, sess.ID, SenderTypeVisitor, "v1", ContentTypeText, "hello")
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if msg.Content != "hello" || msg.ContentType != ContentTypeText {
		t.Errorf("unexpected message: %+v", msg)
	}

	// empty content
	_, err = svc.SendMessage(ctx, sess.ID, SenderTypeAgent, "a1", ContentTypeText, "")
	if err != ErrEmptyMessage {
		t.Errorf("expected ErrEmptyMessage, got %v", err)
	}
}

func TestIMService_SendMessage_Image(t *testing.T) {
	svc := newTestService(5)
	ctx := context.Background()

	ch := createTestChannel(t, svc, 1)
	sess, _ := svc.CreateSession(ctx, CreateSessionInput{TenantID: 1, ChannelID: ch.ID, VisitorID: "v1"})
	_ = svc.AssignAgent(ctx, sess.ID, 200)

	msg, err := svc.SendMessage(ctx, sess.ID, SenderTypeAgent, "a1", ContentTypeImage, "https://example.com/img.png")
	if err != nil {
		t.Fatalf("SendMessage image: %v", err)
	}
	if msg.ContentType != ContentTypeImage {
		t.Errorf("expected image type, got %s", msg.ContentType)
	}

	// closed session rejects messages
	_ = svc.CloseSession(ctx, sess.ID)
	_, err = svc.SendMessage(ctx, sess.ID, SenderTypeVisitor, "v1", ContentTypeText, "late msg")
	if err != ErrSessionClosed {
		t.Errorf("expected ErrSessionClosed, got %v", err)
	}
}

func TestIMService_MaxChatSlots_Exceeded(t *testing.T) {
	svc := newTestService(2) // max 2 slots
	ctx := context.Background()
	agentID := int64(200)

	ch := createTestChannel(t, svc, 1)

	// fill 2 slots
	for i := 0; i < 2; i++ {
		sess, _ := svc.CreateSession(ctx, CreateSessionInput{TenantID: 1, ChannelID: ch.ID, VisitorID: "v"})
		if err := svc.AssignAgent(ctx, sess.ID, agentID); err != nil {
			t.Fatalf("AssignAgent slot %d: %v", i, err)
		}
	}

	// 3rd should fail
	sess3, _ := svc.CreateSession(ctx, CreateSessionInput{TenantID: 1, ChannelID: ch.ID, VisitorID: "v3"})
	if err := svc.AssignAgent(ctx, sess3.ID, agentID); err != ErrMaxChatSlots {
		t.Errorf("expected ErrMaxChatSlots, got %v", err)
	}

	// transfer to full agent should also fail
	sess4, _ := svc.CreateSession(ctx, CreateSessionInput{TenantID: 1, ChannelID: ch.ID, VisitorID: "v4"})
	_ = svc.AssignAgent(ctx, sess4.ID, 999) // assign to different agent
	if err := svc.TransferSession(ctx, sess4.ID, agentID); err != ErrMaxChatSlots {
		t.Errorf("expected ErrMaxChatSlots on transfer, got %v", err)
	}
}
