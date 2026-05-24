package im

import (
	"context"
	"crypto/sha1"
	"fmt"
	"sort"
	"strings"
	"testing"
)

func newSocialTestService() (*SocialChannelService, *MockIMChannelRepo) {
	chRepo := NewMockIMChannelRepo()
	sessRepo := NewMockIMSessionRepo()
	msgRepo := NewMockIMMessageRepo()
	cfgRepo := NewMockSocialChannelConfigRepo()
	svc := NewSocialChannelService(cfgRepo, chRepo, sessRepo, msgRepo)
	return svc, chRepo
}

func createSocialTestChannel(chRepo *MockIMChannelRepo, channelType ChannelType) *IMChannel {
	ch := &IMChannel{
		ID:          1001,
		TenantID:    1,
		ChannelType: channelType,
		Name:        "test-channel",
		Status:      ChannelStatusActive,
	}
	chRepo.Create(context.Background(), ch)
	return ch
}

func computeWeChatSig(token, timestamp, nonce string) string {
	strs := []string{token, timestamp, nonce}
	sort.Strings(strs)
	h := sha1.New()
	h.Write([]byte(strings.Join(strs, "")))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func TestSocialConfig_Create_Success(t *testing.T) {
	svc, chRepo := newSocialTestService()
	createSocialTestChannel(chRepo, ChannelTypeWeChat)

	cfg, err := svc.CreateConfig(context.Background(), CreateSocialConfigInput{
		TenantID:  1,
		ChannelID: 1001,
		Platform:  PlatformWeChat,
		AppID:     "wx1234567890",
		AppSecret: "secret123",
		Token:     "verify_token",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Platform != PlatformWeChat {
		t.Fatalf("expected wechat, got %s", cfg.Platform)
	}
	if cfg.IsVerified {
		t.Fatal("expected not verified initially")
	}
}

func TestSocialConfig_Create_InvalidPlatform(t *testing.T) {
	svc, chRepo := newSocialTestService()
	createSocialTestChannel(chRepo, ChannelTypeWeChat)

	_, err := svc.CreateConfig(context.Background(), CreateSocialConfigInput{
		TenantID:  1,
		ChannelID: 1001,
		Platform:  "telegram",
		AppID:     "test",
		AppSecret: "secret",
		Token:     "token",
	})
	if err != ErrSocialInvalidPlatform {
		t.Fatalf("expected ErrSocialInvalidPlatform, got %v", err)
	}
}

func TestSocialConfig_Create_EmptyAppID(t *testing.T) {
	svc, chRepo := newSocialTestService()
	createSocialTestChannel(chRepo, ChannelTypeWeChat)

	_, err := svc.CreateConfig(context.Background(), CreateSocialConfigInput{
		TenantID:  1,
		ChannelID: 1001,
		Platform:  PlatformWeChat,
		AppID:     "",
		AppSecret: "secret",
		Token:     "token",
	})
	if err != ErrSocialAppIDEmpty {
		t.Fatalf("expected ErrSocialAppIDEmpty, got %v", err)
	}
}

func TestSocialConfig_Create_EmptySecret(t *testing.T) {
	svc, chRepo := newSocialTestService()
	createSocialTestChannel(chRepo, ChannelTypeWeChat)

	_, err := svc.CreateConfig(context.Background(), CreateSocialConfigInput{
		TenantID:  1,
		ChannelID: 1001,
		Platform:  PlatformWeChat,
		AppID:     "wx123",
		AppSecret: "",
		Token:     "token",
	})
	if err != ErrSocialAppSecretEmpty {
		t.Fatalf("expected ErrSocialAppSecretEmpty, got %v", err)
	}
}

func TestSocialConfig_Create_EmptyToken(t *testing.T) {
	svc, chRepo := newSocialTestService()
	createSocialTestChannel(chRepo, ChannelTypeWeChat)

	_, err := svc.CreateConfig(context.Background(), CreateSocialConfigInput{
		TenantID:  1,
		ChannelID: 1001,
		Platform:  PlatformWeChat,
		AppID:     "wx123",
		AppSecret: "secret",
		Token:     "",
	})
	if err != ErrSocialTokenEmpty {
		t.Fatalf("expected ErrSocialTokenEmpty, got %v", err)
	}
}

func TestSocialConfig_Create_ChannelNotFound(t *testing.T) {
	svc, _ := newSocialTestService()

	_, err := svc.CreateConfig(context.Background(), CreateSocialConfigInput{
		TenantID:  1,
		ChannelID: 9999,
		Platform:  PlatformWeChat,
		AppID:     "wx123",
		AppSecret: "secret",
		Token:     "token",
	})
	if err != ErrChannelNotFound {
		t.Fatalf("expected ErrChannelNotFound, got %v", err)
	}
}

func TestWeChat_SignatureVerification(t *testing.T) {
	svc, _ := newSocialTestService()

	token := "test_token"
	timestamp := "1619000000"
	nonce := "abc123"

	// Empty signature should fail
	if svc.VerifyWeChatSignature(token, timestamp, nonce, "") {
		t.Fatal("empty signature should not pass")
	}

	// Wrong signature should fail
	if svc.VerifyWeChatSignature(token, timestamp, nonce, "wrong_sig") {
		t.Fatal("wrong signature should not pass")
	}

	// Correct signature should pass
	correctSig := computeWeChatSig(token, timestamp, nonce)
	if !svc.VerifyWeChatSignature(token, timestamp, nonce, correctSig) {
		t.Fatal("valid signature should pass verification")
	}
}

func TestWeibo_SignatureVerification(t *testing.T) {
	svc, _ := newSocialTestService()

	appSecret := "my_secret"
	body := `{"text":"hello"}`

	h := sha1.New()
	h.Write([]byte(appSecret + body))
	correctSig := fmt.Sprintf("%x", h.Sum(nil))

	if svc.VerifyWeiboSignature(appSecret, body, "wrong") {
		t.Fatal("wrong signature should fail")
	}
	if !svc.VerifyWeiboSignature(appSecret, body, correctSig) {
		t.Fatal("correct signature should pass")
	}
}

func TestSocial_ProcessInboundMessage_Success(t *testing.T) {
	svc, chRepo := newSocialTestService()
	createSocialTestChannel(chRepo, ChannelTypeWeChat)

	sess, msg, err := svc.ProcessInboundMessage(context.Background(), 1001, SocialMessage{
		Platform:    PlatformWeChat,
		OpenID:      "oWx_user123",
		ContentType: ContentTypeText,
		Content:     "你好，我想咨询一下",
		MsgID:       "msg001",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sess.Status != SessionStatusWaiting {
		t.Fatalf("expected waiting, got %s", sess.Status)
	}
	if sess.VisitorID != "oWx_user123" {
		t.Fatalf("expected oWx_user123, got %s", sess.VisitorID)
	}
	if msg.Content != "你好，我想咨询一下" {
		t.Fatalf("unexpected content: %s", msg.Content)
	}
	if msg.SenderType != SenderTypeVisitor {
		t.Fatalf("expected visitor sender, got %s", msg.SenderType)
	}
}

func TestSocial_ProcessInboundMessage_EmptyOpenID(t *testing.T) {
	svc, chRepo := newSocialTestService()
	createSocialTestChannel(chRepo, ChannelTypeWeChat)

	_, _, err := svc.ProcessInboundMessage(context.Background(), 1001, SocialMessage{
		Platform: PlatformWeChat,
		OpenID:   "",
		Content:  "hello",
	})
	if err != ErrSocialOpenIDEmpty {
		t.Fatalf("expected ErrSocialOpenIDEmpty, got %v", err)
	}
}

func TestSocial_ProcessInboundMessage_EmptyContent(t *testing.T) {
	svc, chRepo := newSocialTestService()
	createSocialTestChannel(chRepo, ChannelTypeWeChat)

	_, _, err := svc.ProcessInboundMessage(context.Background(), 1001, SocialMessage{
		Platform: PlatformWeChat,
		OpenID:   "user1",
		Content:  "",
		MediaURL: "",
	})
	if err != ErrSocialMessageEmpty {
		t.Fatalf("expected ErrSocialMessageEmpty, got %v", err)
	}
}

func TestSocial_ProcessInboundMessage_DisabledChannel(t *testing.T) {
	svc, chRepo := newSocialTestService()
	ch := createSocialTestChannel(chRepo, ChannelTypeWeChat)
	ch.Status = ChannelStatusDisabled
	chRepo.Update(context.Background(), ch)

	_, _, err := svc.ProcessInboundMessage(context.Background(), 1001, SocialMessage{
		Platform: PlatformWeChat,
		OpenID:   "user1",
		Content:  "hello",
	})
	if err != ErrChannelDisabled {
		t.Fatalf("expected ErrChannelDisabled, got %v", err)
	}
}

func TestSocial_ProcessInboundMessage_MediaURL(t *testing.T) {
	svc, chRepo := newSocialTestService()
	createSocialTestChannel(chRepo, ChannelTypeWeChat)

	sess, msg, err := svc.ProcessInboundMessage(context.Background(), 1001, SocialMessage{
		Platform:    PlatformWeChat,
		OpenID:      "user1",
		ContentType: ContentTypeImage,
		MediaURL:    "https://wx.qq.com/image/123.jpg",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sess == nil {
		t.Fatal("expected session")
	}
	if msg.ContentType != ContentTypeImage {
		t.Fatalf("expected image, got %s", msg.ContentType)
	}
	if msg.Content != "https://wx.qq.com/image/123.jpg" {
		t.Fatalf("expected media URL as content, got %s", msg.Content)
	}
}

func TestSocial_MarkVerified(t *testing.T) {
	svc, chRepo := newSocialTestService()
	createSocialTestChannel(chRepo, ChannelTypeWeChat)

	cfg, _ := svc.CreateConfig(context.Background(), CreateSocialConfigInput{
		TenantID:  1,
		ChannelID: 1001,
		Platform:  PlatformWeChat,
		AppID:     "wx123",
		AppSecret: "secret",
		Token:     "token",
	})
	if cfg.IsVerified {
		t.Fatal("expected not verified initially")
	}

	err := svc.MarkVerified(context.Background(), 1001)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, _ := svc.GetConfig(context.Background(), 1001)
	if !updated.IsVerified {
		t.Fatal("expected verified after MarkVerified")
	}
}

func TestSocial_Weibo_CreateConfig(t *testing.T) {
	svc, chRepo := newSocialTestService()
	ch := &IMChannel{
		ID:          2001,
		TenantID:    1,
		ChannelType: ChannelTypeWeibo,
		Name:        "weibo-channel",
		Status:      ChannelStatusActive,
	}
	chRepo.Create(context.Background(), ch)

	cfg, err := svc.CreateConfig(context.Background(), CreateSocialConfigInput{
		TenantID:  1,
		ChannelID: 2001,
		Platform:  PlatformWeibo,
		AppID:     "wb_app_key",
		AppSecret: "wb_secret",
		Token:     "wb_token",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Platform != PlatformWeibo {
		t.Fatalf("expected weibo, got %s", cfg.Platform)
	}
}

func TestSocial_ChannelTypes_Registered(t *testing.T) {
	if !validChannelTypes[ChannelTypeWeChat] {
		t.Fatal("wechat should be a valid channel type")
	}
	if !validChannelTypes[ChannelTypeWeibo] {
		t.Fatal("weibo should be a valid channel type")
	}
}
