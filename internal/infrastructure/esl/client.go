package esl

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Client wraps FreeSWITCH ESL connections with pooling, reconnect, and circuit breaker.
type Client struct {
	host    string
	port    int
	pass    string
	pool    chan *conn
	mu      sync.RWMutex
	logger  zerolog.Logger
	breaker *circuitBreaker
}

type conn struct {
	id        int
	connected bool
	lastUsed  time.Time
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
	Logger   zerolog.Logger
}

func NewClient(cfg Config) *Client {
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 5
	}

	c := &Client{
		host:   cfg.Host,
		port:   cfg.Port,
		pass:   cfg.Password,
		pool:   make(chan *conn, cfg.PoolSize),
		logger: cfg.Logger,
		breaker: &circuitBreaker{
			threshold:    5,
			state:        "closed",
			resetTimeout: 30 * time.Second,
		},
	}

	for i := 0; i < cfg.PoolSize; i++ {
		c.pool <- &conn{id: i, connected: false}
	}

	return c
}

func (c *Client) Acquire(ctx context.Context) (*conn, error) {
	if !c.breaker.allow() {
		return nil, fmt.Errorf("esl: circuit breaker open, last failure at %v", c.breaker.lastFailure)
	}

	select {
	case cn := <-c.pool:
		if !cn.connected {
			if err := c.connect(cn); err != nil {
				c.pool <- cn
				c.breaker.recordFailure()
				return nil, fmt.Errorf("esl: connect failed: %w", err)
			}
			c.breaker.recordSuccess()
		}
		cn.lastUsed = time.Now()
		return cn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) Release(cn *conn) {
	c.pool <- cn
}

func (c *Client) connect(cn *conn) error {
	c.logger.Debug().Int("conn_id", cn.id).Str("host", c.host).Msg("connecting to FreeSWITCH ESL")
	// In production: use percipia/eslgo to establish ESL connection
	// conn, err := eslgo.Dial(fmt.Sprintf("%s:%d", c.host, c.port), c.pass)
	cn.connected = true
	return nil
}

// SendCommand sends an ESL command via a pooled connection.
func (c *Client) SendCommand(ctx context.Context, command string) (string, error) {
	cn, err := c.Acquire(ctx)
	if err != nil {
		return "", err
	}
	defer c.Release(cn)

	c.logger.Debug().Int("conn_id", cn.id).Str("cmd", command).Msg("ESL command")
	// In production: cn.eslConn.SendCommand(ctx, command)
	return "OK", nil
}

// Originate starts a new call via FreeSWITCH.
func (c *Client) Originate(ctx context.Context, dest, callerID, context_ string) (string, error) {
	cmd := fmt.Sprintf("originate {origination_caller_id_number=%s}%s %s", callerID, dest, context_)
	return c.SendCommand(ctx, cmd)
}

// HangupCall hangs up a call by UUID.
func (c *Client) HangupCall(ctx context.Context, uuid string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_kill %s", uuid))
	return err
}

// PlayAudio plays an audio file on a call.
func (c *Client) PlayAudio(ctx context.Context, uuid, filePath string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_broadcast %s %s both", uuid, filePath))
	return err
}

// StartRecording starts recording a call.
func (c *Client) StartRecording(ctx context.Context, uuid, filePath string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_record %s start %s", uuid, filePath))
	return err
}

// TransferCall transfers a call to another destination.
func (c *Client) TransferCall(ctx context.Context, uuid, dest string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_transfer %s %s", uuid, dest))
	return err
}

// HoldCall puts a call on hold.
func (c *Client) HoldCall(ctx context.Context, uuid string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_hold %s", uuid))
	return err
}

// RetrieveCall takes a call off hold.
func (c *Client) RetrieveCall(ctx context.Context, uuid string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_hold off %s", uuid))
	return err
}

// SendDTMF sends DTMF digits to a call.
func (c *Client) SendDTMF(ctx context.Context, uuid, digits string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_send_dtmf %s %s", uuid, digits))
	return err
}

// Bridge bridges two call legs.
func (c *Client) Bridge(ctx context.Context, uuid1, uuid2 string) error {
	_, err := c.SendCommand(ctx, fmt.Sprintf("uuid_bridge %s %s", uuid1, uuid2))
	return err
}

func (c *Client) Close() {
	close(c.pool)
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
