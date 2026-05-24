# CCC — Cloud Contact Center

Multi-tenant cloud contact center platform with omnichannel support (Voice, IM, Social, Email, WebChat), AI-assisted agent tooling, and visual IVR orchestration.

## Architecture

```
web/                     React + TypeScript + Ant Design frontend
cmd/server/              Go entrypoint & dependency injection
internal/
  domain/                Business logic (call, campaign, identity, ai, ticket, crm, ...)
  application/           Orchestration (IVR engine, dialer, dashboard hub, IM hub, ...)
  infrastructure/        External integrations (MySQL, Redis, ESL, LLM, WeChat, ...)
  interfaces/http/       Chi router, handlers, middleware
migrations/              MySQL schema migrations
deploy/                  Docker configs (FreeSWITCH, Prometheus, Grafana)
```

**Key stats:** ~85 tables, 160+ API endpoints, 51 handlers, 11 domain services, 243 Go files.

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.25+, Chi router, JWT (golang-jwt/v5), bcrypt |
| Database | MySQL 8.0 (sqlx), Redis 7 |
| Frontend | React 19, TypeScript, Ant Design 5, Zustand |
| Telephony | FreeSWITCH via ESL (Event Socket Library), TCP connection pool |
| AI/LLM | Aliyun DashScope (Qwen), Aliyun NLS (ASR/TTS) |
| Storage | MinIO (S3-compatible, recordings) |
| Messaging | NATS JetStream, Kafka |
| Monitoring | Prometheus + Grafana |
| Deployment | Docker Compose, multi-stage builds |

## Quick Start

### Prerequisites

- Go 1.25+
- Node.js 20+
- Docker & Docker Compose

### Development

```bash
# Clone and configure
cp .env.example .env
# Edit .env with your credentials

# Start infrastructure (MySQL, Redis, MinIO, FreeSWITCH, NATS, Prometheus, Grafana)
make docker-up

# Run database migrations
make migrate-up

# Start backend (with hot reload if air is installed)
make dev

# Start frontend (in another terminal)
make web-install
make web-dev
```

### Production Build

```bash
make build        # Go binary → bin/ccc-server
make web-build    # React → web/dist/
make docker-build # Docker images for all services
```

## Key Features

### Voice
- Inbound/Outbound call management with FreeSWITCH ESL integration
- Visual IVR builder with 20 node types (menu, queue, transfer, record, ASR, TTS, ...)
- Attended & blind transfer, hold/retrieve, DTMF, conference
- Campaign dialer (preview, progressive, predictive, power modes)
- B2B (back-to-back) calls, callback requests
- Call recording with MinIO storage

### Omnichannel
- IM channels (WebChat widget, custom integrations)
- Social channels (WeChat, Weibo)
- Email inbound processing
- Unified session management

### AI/LLM
- Real-time ASR transcription (Aliyun NLS)
- TTS synthesis (Aliyun NLS)
- LLM-powered quality analysis (Qwen via DashScope)
- AI-assisted agent responses
- Conversation analytics, sentiment analysis
- Voice cloning, full-duplex analysis
- Digital employees (autonomous AI agents)

### Agent Workspace
- Real-time agent presence management (Ready, Busy, ACW, Break)
- Skill group routing with priority queuing
- Screen pop (customer lookup on incoming calls)
- Quick replies, agent scripts, knowledge base
- WebRTC quality monitoring (MOS scores)

### Management
- Multi-tenant with tenant isolation (JWT-based)
- Customer CRM with custom fields
- Ticket system with templates and workflows
- CSAT surveys
- Reports & dashboards (agent, call, campaign metrics)
- Audit logging

### WebSocket Real-time
- `/ws/dashboard` — live dashboard metrics
- `/ws/im` — IM message streaming
- `/ws/agent-events` — agent state & call events
- `/ws/transcript` — real-time call transcription

## API Overview

All API endpoints are under `/api/v1/` with JWT authentication (except `/auth/login` and `/widget/*`).

```
POST   /auth/login                    # JWT login
GET    /me/profile                    # Current user profile

# Core entities
CRUD   /tenants, /users, /agents, /skill-groups
CRUD   /calls, /campaigns, /tickets, /customers
CRUD   /ivr-flows, /routing-rules, /phone-numbers

# Configuration
CRUD   /break-reasons, /disposition-codes, /call-tags
CRUD   /audio-files, /business-hours, /quick-replies
GET/PUT /tenant-settings

# AI & Analysis
POST   /ai/analysis/realtime
POST   /ai/voice-clone/tasks
POST   /ai/conversation-analytics/analyze
POST   /ai/training/*

# Real-time
GET    /supervisor/active-calls
GET    /screen-pop/lookup?phone=...
GET    /campaigns/preview/current
```

See `internal/interfaces/http/router.go` for the full route list.

## Configuration

All configuration via environment variables. See `.env.example` for the complete list.

Key variables:

| Variable | Description |
|----------|-------------|
| `DATABASE_DSN` | MySQL connection string |
| `REDIS_ADDR` | Redis address |
| `JWT_SECRET` | JWT signing secret |
| `ESL_HOST/PORT/PASSWORD` | FreeSWITCH ESL connection |
| `ALIYUN_ACCESS_KEY_ID/SECRET` | Aliyun API credentials |
| `DASHSCOPE_API_KEY` | Tongyi Qwen LLM API key |
| `CORS_ALLOW_ORIGIN` | CORS origin (default: `*`) |

## Testing

```bash
make test    # go test -race ./...
make vet     # go vet ./...
make lint    # go vet + golangci-lint (if available)
```

## Load Testing

K6 scripts are available in `tests/k6/`:

```bash
make k6-inbound     # Inbound call simulation
make k6-outbound    # Outbound campaign simulation
make k6-mixed       # Mixed workload
make k6-websocket   # WebSocket connections
make k6-report      # Report query load
```

## License

Proprietary.
