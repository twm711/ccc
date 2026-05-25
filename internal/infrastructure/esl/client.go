package esl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

// Client wraps FreeSWITCH ESL connections with pooling, reconnect, and circuit breaker.
type Client struct {
	host     string
	port     int
	pass     string
	pool     chan *conn
	mu       sync.RWMutex
	logger   zerolog.Logger
	breaker  *circuitBreaker
	closed   int32 // atomic: 1 = closed
	maxIdle  time.Duration
	poolSize int32 // atomic: current pool capacity
	minPool  int
	maxPool  int
	nextID   int32 // atomic: monotonic conn ID
}

type conn struct {
	id        int
	connected bool
	lastUsed  time.Time
	tcpConn   net.Conn
	reader    *bufio.Reader
	mu        sync.Mutex
}

type circuitBreaker struct {
	mu           sync.Mutex
	failures     int
	threshold    int
	state        string // closed, open, half_open
	lastFailure  time.Time
	resetTimeout time.Duration
}

type Config struct {
	Host     string
	Port     int
	Password string
	PoolSize int
	MinPool  int
	MaxPool  int
	Logger   zerolog.Logger
}

func NewClient(cfg Config) *Client {
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 5
	}
	if cfg.MinPool <= 0 {
		cfg.MinPool = cfg.PoolSize
	}
	if cfg.MaxPool <= 0 {
		cfg.MaxPool = cfg.PoolSize * 4
	}
	if cfg.MaxPool < cfg.PoolSize {
		cfg.MaxPool = cfg.PoolSize
	}

	c := &Client{
		host:    cfg.Host,
		port:    cfg.Port,
		pass:    cfg.Password,
		pool:    make(chan *conn, cfg.MaxPool),
		logger:  cfg.Logger,
		maxIdle: 5 * time.Minute,
		minPool: cfg.MinPool,
		maxPool: cfg.MaxPool,
		breaker: &circuitBreaker{
			threshold:    5,
			state:        "closed",
			resetTimeout: 30 * time.Second,
		},
	}
	atomic.StoreInt32(&c.poolSize, int32(cfg.PoolSize))
	atomic.StoreInt32(&c.nextID, int32(cfg.PoolSize))

	for i := 0; i < cfg.PoolSize; i++ {
		c.pool <- &conn{id: i, connected: false}
	}

	return c
}

// Grow adds n connections to the pool up to maxPool.
func (c *Client) Grow(n int) int {
	added := 0
	for i := 0; i < n; i++ {
		cur := atomic.LoadInt32(&c.poolSize)
		if int(cur) >= c.maxPool {
			break
		}
		if !atomic.CompareAndSwapInt32(&c.poolSize, cur, cur+1) {
			continue
		}
		id := int(atomic.AddInt32(&c.nextID, 1) - 1)
		c.pool <- &conn{id: id, connected: false}
		added++
	}
	if added > 0 {
		c.logger.Info().Int("added", added).Int32("total", atomic.LoadInt32(&c.poolSize)).Msg("esl pool: grow")
	}
	return added
}

// Shrink removes n idle connections from the pool down to minPool.
func (c *Client) Shrink(n int) int {
	removed := 0
	for i := 0; i < n; i++ {
		cur := atomic.LoadInt32(&c.poolSize)
		if int(cur) <= c.minPool {
			break
		}
		select {
		case cn := <-c.pool:
			if cn != nil && cn.tcpConn != nil {
				cn.tcpConn.Close()
			}
			atomic.AddInt32(&c.poolSize, -1)
			removed++
		default:
			return removed
		}
	}
	if removed > 0 {
		c.logger.Info().Int("removed", removed).Int32("total", atomic.LoadInt32(&c.poolSize)).Msg("esl pool: shrink")
	}
	return removed
}

// PoolSize returns the current pool capacity.
func (c *Client) PoolSize() int {
	return int(atomic.LoadInt32(&c.poolSize))
}

func (c *Client) Acquire(ctx context.Context) (*conn, error) {
	if atomic.LoadInt32(&c.closed) == 1 {
		return nil, fmt.Errorf("esl: client closed")
	}
	if !c.breaker.allow() {
		return nil, fmt.Errorf("esl: circuit breaker open, last failure at %v", c.breaker.lastFailure)
	}

	select {
	case cn := <-c.pool:
		if cn == nil {
			return nil, fmt.Errorf("esl: client closed")
		}
		// Reconnect if disconnected or idle too long
		if !cn.connected || (c.maxIdle > 0 && time.Since(cn.lastUsed) > c.maxIdle) {
			if cn.connected {
				c.logger.Debug().Int("conn_id", cn.id).Msg("recycling stale ESL connection")
				cn.connected = false
			}
			if err := c.connect(cn); err != nil {
				c.pool <- cn
				c.breaker.recordFailure()
				return nil, fmt.Errorf("esl: connect failed: %w", err)
			}
			c.breaker.recordSuccess()
		}
		cn.lastUsed = time.Now()

		// Auto-grow: if pool is < 20% available, grow by 2 (non-blocking).
		poolCap := int(atomic.LoadInt32(&c.poolSize))
		available := len(c.pool)
		if poolCap > 0 && available*5 < poolCap {
			go c.Grow(2)
		}

		return cn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) Release(cn *conn) {
	if atomic.LoadInt32(&c.closed) == 1 {
		if cn.tcpConn != nil {
			cn.tcpConn.Close()
		}
		return
	}
	c.pool <- cn
}

func (c *Client) connect(cn *conn) error {
	if cn.tcpConn != nil {
		cn.tcpConn.Close()
		cn.tcpConn = nil
		cn.reader = nil
	}

	addr := net.JoinHostPort(c.host, strconv.Itoa(c.port))
	c.logger.Debug().Int("conn_id", cn.id).Str("host", addr).Msg("connecting to FreeSWITCH ESL")

	tcpConn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("esl: dial %s: %w", addr, err)
	}

	reader := bufio.NewReader(tcpConn)

	headers, err := readHeaders(reader)
	if err != nil {
		tcpConn.Close()
		return fmt.Errorf("esl: read auth request: %w", err)
	}
	if headers["Content-Type"] != "auth/request" {
		tcpConn.Close()
		return fmt.Errorf("esl: expected auth/request, got %s", headers["Content-Type"])
	}

	if _, err := fmt.Fprintf(tcpConn, "auth %s\n\n", c.pass); err != nil {
		tcpConn.Close()
		return fmt.Errorf("esl: send auth: %w", err)
	}

	headers, err = readHeaders(reader)
	if err != nil {
		tcpConn.Close()
		return fmt.Errorf("esl: read auth reply: %w", err)
	}
	if !strings.HasPrefix(headers["Reply-Text"], "+OK") {
		tcpConn.Close()
		return fmt.Errorf("esl: auth rejected: %s", headers["Reply-Text"])
	}

	cn.tcpConn = tcpConn
	cn.reader = reader
	cn.connected = true
	c.logger.Info().Int("conn_id", cn.id).Str("host", addr).Msg("ESL connection established")
	return nil
}

// SendCommand sends an ESL command via a pooled connection.
func (c *Client) SendCommand(ctx context.Context, command string) (string, error) {
	cn, err := c.Acquire(ctx)
	if err != nil {
		return "", err
	}
	defer c.Release(cn)

	cn.mu.Lock()
	defer cn.mu.Unlock()

	if deadline, ok := ctx.Deadline(); ok {
		cn.tcpConn.SetDeadline(deadline)
	} else {
		cn.tcpConn.SetDeadline(time.Now().Add(30 * time.Second))
	}
	defer cn.tcpConn.SetDeadline(time.Time{})

	c.logger.Debug().Int("conn_id", cn.id).Str("cmd", command).Msg("ESL command")

	if _, err := fmt.Fprintf(cn.tcpConn, "api %s\n\n", command); err != nil {
		c.breaker.recordFailure()
		cn.connected = false
		return "", fmt.Errorf("esl: send: %w", err)
	}

	headers, err := readHeaders(cn.reader)
	if err != nil {
		c.breaker.recordFailure()
		cn.connected = false
		return "", fmt.Errorf("esl: read response: %w", err)
	}

	var body string
	if cl := headers["Content-Length"]; cl != "" {
		length, _ := strconv.Atoi(cl)
		if length > 0 {
			buf := make([]byte, length)
			if _, err := io.ReadFull(cn.reader, buf); err != nil {
				c.breaker.recordFailure()
				cn.connected = false
				return "", fmt.Errorf("esl: read body: %w", err)
			}
			body = string(buf)
		}
	}

	c.breaker.recordSuccess()
	body = strings.TrimSpace(body)
	if strings.HasPrefix(body, "-ERR") {
		return "", fmt.Errorf("esl: %s", body)
	}
	return body, nil
}

// sanitizeParam rejects ESL command parameters that contain control characters
// which could terminate or inject additional commands.
func sanitizeParam(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\x00' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Originate starts a new call via FreeSWITCH.
func (c *Client) Originate(ctx context.Context, dest, callerID, context_ string) (string, error) {
	cmd := fmt.Sprintf("originate {origination_caller_id_number=%s}%s %s", sanitizeParam(callerID), sanitizeParam(dest), sanitizeParam(context_))
	return c.SendCommand(ctx, cmd)
}

// HangupCall hangs up a call by UUID.
func (c *Client) HangupCall(ctx context.Context, uuid string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_kill %s", sanitizeParam(uuid)))
	return err
}

// PlayAudio plays an audio file on a call.
func (c *Client) PlayAudio(ctx context.Context, uuid, filePath string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_broadcast %s %s both", sanitizeParam(uuid), sanitizeParam(filePath)))
	return err
}

// StartRecording starts recording a call.
func (c *Client) StartRecording(ctx context.Context, uuid, filePath string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_record %s start %s", sanitizeParam(uuid), sanitizeParam(filePath)))
	return err
}

// TransferCall transfers a call to another destination.
func (c *Client) TransferCall(ctx context.Context, uuid, dest string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_transfer %s %s", sanitizeParam(uuid), sanitizeParam(dest)))
	return err
}

// HoldCall puts a call on hold.
func (c *Client) HoldCall(ctx context.Context, uuid string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_hold %s", sanitizeParam(uuid)))
	return err
}

// RetrieveCall takes a call off hold.
func (c *Client) RetrieveCall(ctx context.Context, uuid string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_hold off %s", sanitizeParam(uuid)))
	return err
}

// SendDTMF sends DTMF digits to a call.
func (c *Client) SendDTMF(ctx context.Context, uuid, digits string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_send_dtmf %s %s", sanitizeParam(uuid), sanitizeParam(digits)))
	return err
}

// Bridge bridges two call legs.
func (c *Client) Bridge(ctx context.Context, uuid1, uuid2 string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_bridge %s %s", sanitizeParam(uuid1), sanitizeParam(uuid2)))
	return err
}

// Conference adds a call leg to a conference room via mod_conference.
func (c *Client) Conference(ctx context.Context, uuid, confName string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_transfer %s conference:%s@default inline", sanitizeParam(uuid), sanitizeParam(confName)))
	return err
}

// Eavesdrop starts monitoring a call (listen-only mode).
func (c *Client) Eavesdrop(ctx context.Context, spyUUID, targetUUID string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_transfer %s eavesdrop:%s inline", sanitizeParam(spyUUID), sanitizeParam(targetUUID)))
	return err
}

// EavesdropWhisper monitors with whisper (spy can talk to agent only).
func (c *Client) EavesdropWhisper(ctx context.Context, spyUUID, targetUUID string) error {
	cmd := fmt.Sprintf("uuid_setvar %s eavesdrop_whisper_bleg true", sanitizeParam(spyUUID))
	if _, err := c.SendCommand(ctx, cmd); err != nil {
		return err
	}
	return c.Eavesdrop(ctx, spyUUID, targetUUID)
}

// EavesdropBarge monitors with barge (spy can talk to both parties).
func (c *Client) EavesdropBarge(ctx context.Context, spyUUID, targetUUID string) error {
	spySafe := sanitizeParam(spyUUID)
	if _, err := c.SendCommand(ctx, fmt.Sprintf("uuid_setvar %s eavesdrop_whisper_aleg true", spySafe)); err != nil {
		return err
	}
	if _, err := c.SendCommand(ctx, fmt.Sprintf("uuid_setvar %s eavesdrop_whisper_bleg true", spySafe)); err != nil {
		return err
	}
	return c.Eavesdrop(ctx, spyUUID, targetUUID)
}

// Intercept takes over a call from another agent.
func (c *Client) Intercept(ctx context.Context, interceptorUUID, targetUUID string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_transfer %s intercept:%s inline", sanitizeParam(interceptorUUID), sanitizeParam(targetUUID)))
	return err
}

// Coach starts a coaching session (coach audio to agent only, customer cannot hear).
// Uses whisper_bleg so the coach is heard by the agent (B-leg) only.
func (c *Client) Coach(ctx context.Context, coachUUID, targetUUID string) error {
	cmd := fmt.Sprintf("uuid_setvar %s eavesdrop_whisper_bleg true", sanitizeParam(coachUUID))
	if _, err := c.SendCommand(ctx, cmd); err != nil {
		return err
	}
	return c.Eavesdrop(ctx, coachUUID, targetUUID)
}

// WhisperAnnouncement plays a whisper announcement to the agent before connecting.
func (c *Client) WhisperAnnouncement(ctx context.Context, uuid, audioFile string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_broadcast %s %s aleg", sanitizeParam(uuid), sanitizeParam(audioFile)))
	return err
}

// RegisterSIPPhone checks if a SIP phone is registered via mod_sofia.
func (c *Client) RegisterSIPPhone(ctx context.Context, extension, password, domain string) error {
	cmd := fmt.Sprintf("sofia_contact */%s@%s", sanitizeParam(extension), sanitizeParam(domain))
	resp, err := c.SendCommand(ctx, cmd)
	if err != nil {
		return err
	}
	if strings.Contains(resp, "error") {
		return fmt.Errorf("esl: extension %s@%s not registered", extension, domain)
	}
	return nil
}

// OriginateToPhone bridges a call to an external phone number (field mode).
func (c *Client) OriginateToPhone(ctx context.Context, uuid, phoneNumber, callerID, gateway string) error {
	dest := fmt.Sprintf("sofia/gateway/%s/%s", sanitizeParam(gateway), sanitizeParam(phoneNumber))
	cmd := fmt.Sprintf("uuid_transfer %s bridge:{origination_caller_id_number=%s}%s inline", sanitizeParam(uuid), sanitizeParam(callerID), dest)
	_, err := c.SendCommand(ctx, cmd)
	return err
}

// OriginateB2B initiates a back-to-back call (双呼) bridging two external parties.
func (c *Client) OriginateB2B(ctx context.Context, callerNum, calleeNum, callerID, gateway string) error {
	dest := fmt.Sprintf("sofia/gateway/%s/%s", sanitizeParam(gateway), sanitizeParam(calleeNum))
	cmd := fmt.Sprintf("originate {origination_caller_id_number=%s}sofia/gateway/%s/%s &bridge(%s)", sanitizeParam(callerID), sanitizeParam(gateway), sanitizeParam(callerNum), dest)
	_, err := c.SendCommand(ctx, cmd)
	return err
}

// FlashSMS sends a flash/push SMS via FreeSWITCH chat API.
func (c *Client) FlashSMS(ctx context.Context, from, to, message string) error {
	cmd := fmt.Sprintf("chat sms|%s|%s|%s", sanitizeParam(from), sanitizeParam(to), sanitizeParam(message))
	_, err := c.SendCommand(ctx, cmd)
	return err
}

// StartHealthCheck periodically sends a status command to verify ESL connectivity.
func (c *Client) StartHealthCheck(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if atomic.LoadInt32(&c.closed) == 1 {
				return
			}
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			if _, err := c.SendCommand(checkCtx, "api status"); err != nil {
				c.logger.Warn().Err(err).Msg("esl health check: ping failed")
			}
			cancel()
		}
	}
}

func (c *Client) Close() {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return
	}
	// Drain pool and close TCP connections.
	// Do NOT close the channel — Release() may still send after this returns.
	// The channel will be GC'd when the Client is no longer referenced.
	for {
		select {
		case cn := <-c.pool:
			if cn != nil && cn.tcpConn != nil {
				cn.tcpConn.Close()
			}
		default:
			return
		}
	}
}

func readHeaders(r *bufio.Reader) (map[string]string, error) {
	headers := make(map[string]string)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if idx := strings.Index(line, ": "); idx > 0 {
			headers[line[:idx]] = line[idx+2:]
		}
	}
	return headers, nil
}

// Circuit breaker methods
func (cb *circuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case "open":
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = "half_open"
			return true
		}
		return false
	default:
		return true
	}
}

func (cb *circuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cb.threshold {
		cb.state = "open"
	}
}

func (cb *circuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = "closed"
}
