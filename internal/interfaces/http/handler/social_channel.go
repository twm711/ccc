package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/divord97/ccc/internal/domain/im"
	"github.com/divord97/ccc/pkg/response"
	"github.com/go-chi/chi/v5"
)

type SocialChannelHandler struct {
	svc *im.SocialChannelService
}

func NewSocialChannelHandler(svc *im.SocialChannelService) *SocialChannelHandler {
	return &SocialChannelHandler{svc: svc}
}

// CreateConfig creates a social channel config (WeChat/Weibo credentials).
func (h *SocialChannelHandler) CreateConfig(w http.ResponseWriter, r *http.Request) {
	var in im.CreateSocialConfigInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, 400, err.Error())
		return
	}
	cfg, err := h.svc.CreateConfig(r.Context(), in)
	if err != nil {
		response.Error(w, 422, err.Error())
		return
	}
	response.JSON(w, 201, cfg)
}

// GetConfig returns social config for a channel.
func (h *SocialChannelHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	channelID, _ := strconv.ParseInt(chi.URLParam(r, "channelID"), 10, 64)
	cfg, err := h.svc.GetConfig(r.Context(), channelID)
	if err != nil {
		response.Error(w, 404, err.Error())
		return
	}
	response.JSON(w, 200, cfg)
}

// DeleteConfig removes a social channel config.
func (h *SocialChannelHandler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.svc.DeleteConfig(r.Context(), id); err != nil {
		response.Error(w, 500, err.Error())
		return
	}
	w.WriteHeader(204)
}

// WeChatVerify handles WeChat server URL verification (GET request).
// WeChat sends: signature, timestamp, nonce, echostr.
func (h *SocialChannelHandler) WeChatVerify(w http.ResponseWriter, r *http.Request) {
	channelID, _ := strconv.ParseInt(chi.URLParam(r, "channelID"), 10, 64)
	sig := r.URL.Query().Get("signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")
	echoStr := r.URL.Query().Get("echostr")

	cfg, err := h.svc.GetConfig(r.Context(), channelID)
	if err != nil {
		http.Error(w, "config not found", 404)
		return
	}

	if !h.svc.VerifyWeChatSignature(cfg.Token, timestamp, nonce, sig) {
		http.Error(w, "invalid signature", 403)
		return
	}

	h.svc.MarkVerified(r.Context(), channelID)
	w.WriteHeader(200)
	w.Write([]byte(echoStr))
}

// WeChatReceive handles inbound WeChat messages (POST request).
func (h *SocialChannelHandler) WeChatReceive(w http.ResponseWriter, r *http.Request) {
	channelID, _ := strconv.ParseInt(chi.URLParam(r, "channelID"), 10, 64)

	sig := r.URL.Query().Get("signature")
	timestamp := r.URL.Query().Get("timestamp")
	nonce := r.URL.Query().Get("nonce")

	cfg, err := h.svc.GetConfig(r.Context(), channelID)
	if err != nil {
		http.Error(w, "config not found", 404)
		return
	}
	if !h.svc.VerifyWeChatSignature(cfg.Token, timestamp, nonce, sig) {
		http.Error(w, "invalid signature", 403)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", 400)
		return
	}

	msg, err := parseWeChatXML(body)
	if err != nil {
		http.Error(w, "parse failed", 400)
		return
	}

	sess, imMsg, err := h.svc.ProcessInboundMessage(r.Context(), channelID, msg)
	if err != nil {
		response.Error(w, 500, err.Error())
		return
	}

	response.JSON(w, 200, map[string]interface{}{
		"session": sess,
		"message": imMsg,
	})
}

// WeiboVerify handles Weibo webhook verification.
func (h *SocialChannelHandler) WeiboVerify(w http.ResponseWriter, r *http.Request) {
	echoStr := r.URL.Query().Get("echostr")
	w.WriteHeader(200)
	w.Write([]byte(echoStr))
}

// WeiboReceive handles inbound Weibo messages (POST request).
func (h *SocialChannelHandler) WeiboReceive(w http.ResponseWriter, r *http.Request) {
	channelID, _ := strconv.ParseInt(chi.URLParam(r, "channelID"), 10, 64)

	cfg, err := h.svc.GetConfig(r.Context(), channelID)
	if err != nil {
		http.Error(w, "config not found", 404)
		return
	}

	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", 400)
		return
	}

	weiboSig := r.URL.Query().Get("signature")
	if !h.svc.VerifyWeiboSignature(cfg.AppSecret, string(rawBody), weiboSig) {
		http.Error(w, "invalid signature", 403)
		return
	}

	var payload struct {
		SenderID string `json:"sender_id"`
		Text     string `json:"text"`
		MsgID    string `json:"msg_id"`
	}
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		http.Error(w, "parse failed", 400)
		return
	}

	msg := im.SocialMessage{
		Platform:    im.PlatformWeibo,
		OpenID:      payload.SenderID,
		ContentType: im.ContentTypeText,
		Content:     payload.Text,
		MsgID:       payload.MsgID,
	}

	sess, imMsg, err := h.svc.ProcessInboundMessage(r.Context(), channelID, msg)
	if err != nil {
		response.Error(w, 500, err.Error())
		return
	}

	response.JSON(w, 200, map[string]interface{}{
		"session": sess,
		"message": imMsg,
	})
}

// parseWeChatXML extracts key fields from WeChat XML message payload.
func parseWeChatXML(body []byte) (im.SocialMessage, error) {
	// Simplified XML parsing — production would use encoding/xml struct mapping.
	// WeChat sends XML like: <xml><ToUserName>...</ToUserName><FromUserName>openid</FromUserName>
	//   <MsgType>text</MsgType><Content>hello</Content><MsgId>123</MsgId></xml>
	msg := im.SocialMessage{
		Platform:    im.PlatformWeChat,
		ContentType: im.ContentTypeText,
	}

	msg.OpenID = extractXMLTag(body, "FromUserName")
	msg.MsgID = extractXMLTag(body, "MsgId")

	msgType := extractXMLTag(body, "MsgType")
	switch msgType {
	case "image":
		msg.ContentType = im.ContentTypeImage
		msg.MediaURL = extractXMLTag(body, "PicUrl")
		msg.Content = msg.MediaURL
	case "voice":
		msg.ContentType = im.ContentTypeAudio
		msg.MediaURL = extractXMLTag(body, "MediaId")
		msg.Content = extractXMLTag(body, "Recognition")
	case "video", "shortvideo":
		msg.ContentType = im.ContentTypeVideo
		msg.MediaURL = extractXMLTag(body, "MediaId")
		msg.Content = msg.MediaURL
	default:
		msg.Content = extractXMLTag(body, "Content")
	}

	return msg, nil
}

// extractXMLTag extracts value between <tag>value</tag> or <tag><![CDATA[value]]></tag>.
func extractXMLTag(data []byte, tag string) string {
	s := string(data)
	start := "<" + tag + ">"
	startCDATA := "<" + tag + "><![CDATA["
	end := "</" + tag + ">"

	var idx int
	var val string
	if idx = indexOf(s, startCDATA); idx >= 0 {
		val = s[idx+len(startCDATA):]
		if endIdx := indexOf(val, "]]>"); endIdx >= 0 {
			return val[:endIdx]
		}
	}
	if idx = indexOf(s, start); idx >= 0 {
		val = s[idx+len(start):]
		if endIdx := indexOf(val, end); endIdx >= 0 {
			return val[:endIdx]
		}
	}
	return ""
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
