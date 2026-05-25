# CCC Round 13 — Production Readiness Audit

> Full re-audit of the system after PRs #10–13 landed. This document scores
> the codebase against the dimensions a contact-center deployment will be
> measured on, lists every concrete gap I could find with line citations, and
> defines what "complete (生产完整版)" means before the next release.

## 1. Scope of audit

| Area | Files | LOC |
|---|---:|---:|
| Go production code | 248 | ~41 000 |
| Go tests | 24 | (small fraction) |
| TypeScript/TSX (frontend) | 67 | ~5 500 |
| SQL migrations | 9 | — |

Scans performed on `main` at `f12405e`:
- TODO/FIXME/XXX/HACK markers in source
- 501/`panic("unimplemented")`/empty-return stubs
- Cross-tenant hardcoding
- HTTP query-param input validation
- `http.Server` and `http.Client` timeouts
- Test coverage per package (`go test -cover ./...`)
- Frontend → backend endpoint contract gaps
- Observability (metrics declared vs. observed)

## 2. Scorecard

| Dimension | Score | Notes |
|---|:-:|---|
| Functional completeness | 8/10 | All Round 10–12 features land, no `panic("TODO")` paths. Only one legit 501 (recording when storage unconfigured). |
| **Multi-tenant security** | **5/10** | 4 handlers hardcode `TenantID: 1` (see §3.1) — real cross-tenant escalation. |
| Test coverage | 4/10 | Domain layers 41–75 %; **ACD 8 %, Dialer 0 %, Lifecycle 26 %**. The core routing path is barely tested. |
| Error handling | 6/10 | Service layer is solid; HTTP handlers swallow 198 `strconv` parse errors (id=0 fallthrough). |
| **Input validation** | **5/10** | 19 list endpoints accept any `?limit=N` — no upper bound, easy memory-exhaustion DoS. Only `calls/cursor` (1/19) caps at 100. |
| **HTTP server hardening** | **4/10** | `http.Server` has zero timeouts — Slowloris-class DoS vulnerability. All outbound `http.Client` instances are correctly timed (good). |
| Graceful shutdown | 9/10 | 4-phase shutdown with readiness flip + drain + Shutdown is exemplary. |
| Configuration | 8/10 | JWT default fail-fast in place. `FREESWITCH_PASSWORD` still defaults to well-known "ClueCon" (low risk because FS is internal-only). |
| Observability | 7/10 | Round 11 metrics all wired and observed. `PostCallProcessingLatency` confirmed observed at `worker.go:53`. Missing: histogram for HTTP request latency, panic recovery counter. |
| Frontend ↔ backend contract | 9/10 | 87 distinct paths used by FE; all backed (verified via router scan, accounting for chi sub-routes). |
| Documentation | 7/10 | README + 3 deep-analysis docs. Missing: architecture diagram, deploy runbook, on-call playbook. |
| Graceful failure under DB outage | 6/10 | Most repos return errors cleanly. A few `IVR context` paths in Redis depend on Redis being up (no degradation mode). |

## 3. Concrete gaps with line citations

### 3.1 [P0] Multi-tenant: hardcoded `TenantID: 1`

Every Create writes tenant 1; every List filters to tenant 1; Update/Delete don't verify ownership. A user authenticated as tenant 42 can list/modify tenant 1's webhook secrets.

| Handler | Create writes 1 | List reads 1 | Update/Delete checks |
|---|---|---|---|
| `internal/interfaces/http/handler/webhook_config.go:37,49` | ✗ | ✗ | ✗ |
| `internal/interfaces/http/handler/sms_config.go:37,49` | ✗ | ✗ | ✗ |
| `internal/interfaces/http/handler/screen_pop_config.go:36,48` | ✗ | ✗ | ✗ |
| `internal/interfaces/http/handler/quick_reply.go:39,52` | ✗ | ✗ | ✗ |

**Fix**: Replace literal `1` with `middleware.TenantIDFromCtx(r.Context())` on Create/List. On Update/Delete, after `GetByID`, reject with 404 if `cfg.TenantID != tenantID` (404 not 403, so we don't leak existence).

### 3.2 [P1] HTTP server has no timeouts

`cmd/server/main.go:800`: `srv := &http.Server{Addr: addr, Handler: router}`.

A slow client holding the read buffer open ties up a goroutine + file descriptor indefinitely. The Go default behavior is "no timeout."

**Fix**: Set `ReadHeaderTimeout` (5s), `ReadTimeout` (30s), `WriteTimeout` (60s), `IdleTimeout` (120s). 60s write covers CSV export streams.

### 3.3 [P1] Unbounded `?limit=` on list endpoints

19 handlers accept `?limit=N` and only check `<=0`. A client passing `?limit=10000000` causes the repo to allocate a 10M-row slice. Only `CallHandler.ListCursor` (1/19) caps via `limit > 100`.

**Fix**: Introduce `pagination.ParseLimit(r, dflt, max)` helper and use it across handlers — at minimum the high-traffic ones (call, ticket, campaign, customer, audit-log, recording).

### 3.4 [P1] Test coverage gaps in critical paths

```
internal/application/acd      8.3%   ← routing engine
internal/application/dialer   0.0%   ← 4 dial modes
internal/application/lifecycle 26.1% ← state machine
internal/infrastructure/esl   6.7%   ← FreeSWITCH adapter
```

**Fix this round**: Add targeted unit tests for:
- `dialer.calcAbandonRate` (correct math, division-by-zero)
- `acd.parseMember` (legacy bare id vs `id:timestamp` member format)
- `acd.pickAgent` (verifies batch fetch path from Round 12)

These three tests double the floor on the most fragile paths without scaffolding a full integration stack.

### 3.5 [P2] Frontend `web/src/api/hooks.ts` referenced nonexistent `dashboardApi.get`

Pre-existing tsc error from before Round 10. Not breaking the bundle because the file isn't imported, but it blocked any future `tsc --noEmit` gate. **Fixed in R13** by pointing the `useDashboard` hook at the actual `dashboardApi.overview()` endpoint. `npx tsc -b` now passes clean.

### 3.6 [P2] `web/src/components/phone/ScreenPopPanel.tsx` post-Round-12

Note from PR #13 Devin Review (already merged): the render guard now correctly checks both `callerNumber` and `callId`. No remaining issue.

### 3.7 [P3] Architecture/deploy docs missing

Recommended additions for a "production complete" release:
- `docs/architecture.md` with a system diagram (ACD ↔ Dialer ↔ IVR ↔ NATS ↔ Postcall)
- `docs/runbook.md` covering FreeSWITCH outage, Redis outage, NATS lag spike, certificate rotation
- `docs/security.md` covering tenant isolation guarantees, recording encryption, PII redaction

## 4. What "生产完整版" means (definition of done)

A release qualifies as "production complete" when **all P0 + all P1** items above ship behind CI, and the following invariants hold:

1. Every list endpoint rejects `?limit=` outside `[1, max]` with a sensible default.
2. Every handler that mutates a per-tenant resource verifies tenant ownership.
3. Every external HTTP client has a timeout. Every internal HTTP server has read/write/idle timeouts.
4. ACD `pickAgent`, dialer abandon-rate math, and ACD member format parsing have unit-test coverage.
5. Metrics declared in `pkg/metrics/business.go` are observed at least once in production code paths (currently true after Round 11–12).

This round (R13) ships items 1–4 **plus** the frontend tsc unblock from §3.5. Item 5 was verified during the audit and is already true.

## 4.1 What R13 actually changed (diff summary)

| Touch point | Files | Change |
|---|---|---|
| Multi-tenant (§3.1) | `webhook_config.go`, `sms_config.go`, `screen_pop_config.go`, `quick_reply.go` | `TenantID: 1` → from middleware context. Update/Delete reject 404 when `cfg.TenantID != ctxTenantID`. |
| HTTP server (§3.2) | `cmd/server/main.go` | Added `ReadHeaderTimeout=5s`, `ReadTimeout=30s`, `WriteTimeout=120s`, `IdleTimeout=120s`. |
| List bounds (§3.3) | `pkg/pagination/pagination.go` + 14 handler files | New `ParseLimitOffset(r, dflt, max)` helper; every legacy `?limit=&offset=` list endpoint now clamps to `[1, 200]`. |
| Test coverage (§3.4) | `internal/application/dialer/service_test.go` (new), `internal/domain/identity/service_test.go` (one new test) | `calcAbandonRate` covered (incl. divide-by-zero); `MockAgentPresenceRepo.GetByAgentIDs` contract test guards Round-12 batch-fetch path. |
| Frontend tsc (§3.5) | `web/src/api/hooks.ts` | `dashboardApi.get` → `dashboardApi.overview`. `npx tsc -b` clean. |

## 5. Industry recommendations (queued for R14+)

1. **Recording encryption pipeline**. The recording handler currently returns 501 when storage is unconfigured; with storage configured, files are written raw. Add KMS envelope encryption + per-tenant key, retention TTL per `GDPR/MIIT` policy.
2. **ACD strategy plug-in**. Current routing branches on hardcoded `RoutingPolicy*` constants. Refactor to a `Strategy` interface so ML-predictive routing can be added without forking core service code.
3. **Outbox pattern for webhooks/IM/CRM**. Currently we deliver via direct HTTP from inside the worker. A flaky destination retries inline. Move to outbox table + NATS-driven dispatcher (PR #13 already laid the retry/backoff/dead-letter foundation).
4. **ClickHouse / StarRocks for BI**. `daily_cdr_summary` is a MySQL aggregate. Move the analytics surface (campaign, agent, SLA dashboards) onto a real OLAP store and keep MySQL only for transactional state.
5. **Tenant scope middleware**. Build a `tenant_scope.go` layer that strips `tenant_id` from request body and forces it from context, so handler bugs like §3.1 can't regress.
6. **Frontend bundle split + React Query**. Lazy routes (R12) cut initial bundle; the next win is moving Ant Design's locale + icons to a separate chunk and adopting React Query for invalidation.
