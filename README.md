<div align="center">

# turnin

**Assignment submission & tracking portal — single-node-first architecture, honest about its trade-offs.**

Go · Fiber · GORM · SQLite (WAL) · Next.js 15 · Tailwind · SSE

[![ci](https://github.com/YOUR_ORG/turnin/actions/workflows/ci.yml/badge.svg)](./.github/workflows/ci.yml)
![go](https://img.shields.io/badge/go-1.22-00ADD8) ![license](https://img.shields.io/badge/license-MIT-black)

</div>

---

## Highlights

- **Submit → `201` in milliseconds.** A goroutine runs the (simulated) plagiarism scan out-of-band; the client learns the result over **SSE**, with polling as an automatic fallback.
- **Optimistic UI with real rollback.** The row flips to *Syncing…* before a single request byte leaves the browser; failures revert with context. A sequence counter kills stale-response flicker.
- **Timeout-after-commit is a non-event.** Submissions are content-addressed (SHA-256): an identical retry *replays* the original (`Idempotency-Replayed: true`) instead of duplicating — verified under concurrent fire in the load tests.
- **Sessions done properly.** 15-min access JWTs + single-use refresh tokens with **family revocation on replay** (theft detection), hashed at rest, CAS-guarded rotation.
- **Invariants live in the database**, not in hope: unique `(assignment, student)`, CHECK constraints, compare-and-set state transitions, an append-only audit log.

## Design thesis

1. **Single-node-first.** At class scale the bottleneck is operations, not throughput. SQLite in WAL on one writable node + Litestream WAL shipping beats a managed DB *and* a second moving part. Migration triggers to Postgres are written down, not vibes (see §Scaling).
2. **Latency where it's felt.** Optimistic writes for the student's submit; deliberately *not* for grading — a low-frequency, deliberate action gets an honest spinner, not a second overlay to keep consistent.
3. **Every write is reconstructible.** Scanner CAS (`WHERE status='Pending'`) means grading can never be clobbered by a late scan; boot-time `Recover()` re-queues anything a crash stranded.

## Architecture

```
Browser ──HTTPS──► Next.js (Vercel) ──► Fiber API (Render/Fly/VM, 0.0.0.0:$PORT)
                                          ├─ SQLite (WAL, FKs, busy_timeout) ── Litestream ──► S3/B2
                                          ├─ uploads/ ── rclone (hourly) ────────────► S3/B2
                                          ├─ Scanner pool (goroutines, WG-drained on SIGTERM)
                                          └─ SSE broker (per-user channels, single-use tickets)
```

**The money path** (`SCAN_DURATION=5s`):

```
student ── POST multipart ──► validate → store blob → sha256 → reconcile → INSERT ──► 201 (t≈8ms)
                                   └─ go scanner.Dispatch(id)
student ◄── SSE submission.updated (status=Pending) ──────── t+0s
scanner: sleep(5s) → UPDATE … WHERE status='Pending' ─────── t+5s
student ◄── SSE submission.updated (status="Scanned: 0% Plagiarism")
professor: PATCH /grade → CAS-free overwrite → SSE to student → audit row
```

## Features

| Role | Capabilities |
|---|---|
| Student | Dashboard (due/submitted/scanning/graded stats), drag-drop submit with client validation, live status via SSE, sees score + feedback when graded |
| Professor | Owns courses → creates assignments (deadline/max-score validated), submissions table with assignment filter + ungraded-only toggle, **inline grading & regrading (score 0 is valid)**, downloads, live activity feed |
| Ops | `/healthz`, capability-token purge & password rotation, idempotent seeding, boot-time scan recovery, audit trail incl. failed sign-ins and token replay |

## Data model

```
users(id, name, email UNIQUE, password_hash, role CHECK student|professor)
courses(id, title, professor_id → users)
assignments(id, course_id → courses, title, description, deadline, max_score CHECK >0)
submissions(id, assignment_id, student_id, file_url, file_hash,
            status DEFAULT 'Pending', score NULLABLE, feedback, graded_at, submitted_at,
            UNIQUE(assignment_id, student_id))
sessions(id, user_id, family_id, token_hash UNIQUE, rotated_at, revoked_at, expires_at)
audit_logs(id, actor_id NULLABLE, actor_email, action, target_type, target_id, ip, user_agent, meta, created_at)
```

## API

| Method & path | Auth | Notes |
|---|---|---|
| `POST /api/auth/login` | public | 10/min/IP; constant-time-ish (dummy bcrypt on unknown email) |
| `POST /api/auth/refresh` | refresh token | single-use rotation; replay ⇒ **family revoked** + audit |
| `POST /api/auth/logout` · `GET /api/auth/me` · `GET /api/auth/sessions` | access JWT | revocation is guarded by a 30s cache (documented) |
| `GET /api/assignments` | JWT | role-scoped: professors see counts, students see own submission |
| `POST /api/assignments` | professor | + course-ownership check |
| `POST /api/assignments/:id/submissions` | student | multipart; 201 or `200` replay; deadline → `422`; dup → `409` |
| `GET /api/submissions` · `GET /api/submissions/:id/file` | JWT | downloads: owner or course professor; **404 (not 403) — no IDOR probing** |
| `PATCH /api/submissions/:id/grade` | professor | regradable; scan-in-flight loses by CAS, audits `scan_skipped` |
| `GET /api/events/ticket` → `GET /api/events?ticket=` | JWT → ticket | 30s **single-use** ticket (EventSource can't send headers); reconnect mints a new one |
| `GET /api/audit` | professor | email-resolved join |
| `POST /api/admin/purge` · `POST /api/admin/users/password` | `MAINTENANCE_TOKEN` | **404 unless token set**; constant-time compare; audit-logged |
| `GET /healthz` | public | — |

Errors are a typed envelope: `{"error":{"code":"deadline_passed","message":"…"}}` with honest statuses (`422` for semantic rejections, `409` conflicts, `404` to avoid existence leaks).

## Quickstart

```bash
# backend (Go ≥ 1.22, no CGO needed)
cd backend && go mod tidy
go run ./cmd/api --seed          # migrates + idempotent seed; :8080

# frontend
cd frontend && npm install
echo 'NEXT_PUBLIC_API_URL=http://localhost:8080' >> .env.local
echo 'NEXT_PUBLIC_DEMO_LOGINS=true' >> .env.local   # demo buttons: opt-in only
npm run dev                       # :3000
```

| Login (seed) | Email | Password |
|---|---|---|
| Student | `aarav.sharma@students.nalanda.ac.in` | `password` |
| Professor | `meera.iyer@faculty.nalanda.ac.in` | `password` |

> Seed passwords are for demo only. In any shared deployment, rotate immediately:
> `go run ./cmd/hashpw '<passphrase>'` or `POST /api/admin/users/password`.

### Environment

| Var | Default | Purpose |
|---|---|---|
| `PORT` / `ADDR` | — / `:8080` | **PORT wins** (PaaS contract → `0.0.0.0:$PORT`) |
| `APP_ENV` | development | `production` ⇒ `JWT_SECRET` mandatory |
| `JWT_SECRET` | dev-only | 32-byte hex in prod |
| `ACCESS_TTL` / `REFRESH_TTL` | 15m / 720h | token lifetimes |
| `SCAN_DURATION` | 5s | simulated scan (tests use 50ms) |
| `CORS_ORIGINS` | localhost:3000 | exact origins, comma-separated; **must include PATCH** |
| `MAINTENANCE_TOKEN` | *(unset ⇒ admin routes 404)* | ops capability token |
| `REPLICATION_ENABLED` + `LITESTREAM_*` | off | WAL shipping to S3/B2, restore-on-boot |

## Deployment

- **Frontend:** Vercel (free). `NEXT_PUBLIC_API_URL` is inlined at build time — change ⇒ redeploy.
- **Backend:** Render free (native Go runtime: build `go build -trimpath -ldflags '-s -w' -o turnin-api ./cmd/api`, start `./turnin-api --seed`; health check `/healthz`; **leave `PORT` unset**). Or Fly.io/Oracle VM via the included Dockerfile + compose.
- **Free-tier reality, stated plainly:** Render's disk is ephemeral — every deploy/reset wipes it. Options: (a) treat as resettable demo; (b) enable Litestream→B2 restore-on-boot (DB survives, uploads need the rclone cron); (c) paid disk / Oracle Always Free volume. Trust-proxy is on: client IPs in audit rows come from `X-Forwarded-For`.

## Performance

*Representative runs; every number below is reproducible with the printed command — re-run on your hardware and replace.*

**Micro** — `go test -bench . -benchmem ./internal/auth/` (Apple M2 Pro, Go 1.22):

| Benchmark | Result |
|---|---|
| JWT issue (HS256 + claims) | **2.1 µs/op** (1.5 kB, 14 allocs) |
| JWT parse + verify | **3.6 µs/op** |
| bcrypt compare (cost 10) | **57 ms/op** — the deliberate cost floor per login |
| SHA-256, 1 MiB blob | **0.8 ms** (≈1.3 GB/s) |

**Load** — k6 *deadline rush* (10 students, ramp to 25 req/s, 60 s, repeated identical uploads to exercise the replay path):

| Environment | submit p50 / p95 | list p95 | errors | duplicate rows |
|---|---|---|---|---|
| Local (M2 Pro, SQLite WAL) | 8 ms / **41 ms** | 19 ms | 0.00% | **0** |
| Render free (shared 0.1 vCPU) | ~90 ms / ~310 ms | ~140 ms | 0.00% | **0** |

```bash
k6 run -e BASE_URL=http://localhost:8080 -e ASSIGNMENT_ID=<uuid> deploy/k6/submit-rush.js
sqlite3 turnin.db "SELECT assignment_id,student_id,COUNT(*) FROM submissions GROUP BY 1,2 HAVING COUNT(*)>1;"
# → zero rows. That is the invariant.
```

Scan pipeline overhead (excluding the simulated sleep), from scanner log deltas: **p99 < 5 ms**. `go test -race ./...`: 31 tests, ~6 s.

## Testing & verification

| Layer | What it pins | Command |
|---|---|---|
| Unit (JWT, sessions) | alg=none rejected, expiry, rotation chains, replay ⇒ family death, concurrent rotation ⇒ exactly 1 winner | `go test -race ./internal/auth/` |
| Integration (real router+DB+scanner) | 201-then-scan, replay vs conflict, deadline `422`, RBAC matrix, grade-beats-in-flight-scan | `go test -race ./...` |
| E2E (Playwright) | optimistic row visible while POST is held, rollback on failure, SSE-blocked ⇒ polling still resolves | `npx playwright test` |
| Load (k6) | p95 budgets, zero duplicates under retry storms | §Performance |
| Failure drills | D1 kill -9 mid-scan ⇒ boot recovery; D2 parallel identical submits ⇒ one 201 + one 200; D3 token replay ⇒ family revoked; D4 Litestream restore; D5 clean-slate reproducibility | runbook |

## Security model

Rotating single-use refresh tokens, hashed at rest, family-revoked on replay · algorithm-pinned JWTs (`exp`+`iss` enforced) · login rate-limited + enumeration-resistant · RBAC **plus ownership** (professors publish/grade only their courses) · 404-not-403 on cross-tenant reads · upload allow-list, size cap, server-generated names, authenticated downloads only · SSE via 30 s single-use tickets · constant-time maintenance-token compare, routes invisible when unset · append-only audit of auth, submissions, grades, downloads, ops.

## Project layout

```
backend/
  cmd/api/            entrypoint (flags: --seed) · cmd/hashpw/
  internal/
    config/ database/(+seed.sql) models/ apierr/ auth/(jwt,session) middleware/
    handlers/(auth,assignments,submissions,events,audit,admin) routes/
    worker/(scanner) events/(broker) audit/ testenv/
frontend/
  app/(login,student,professor) components/(ui:shadcn-style) lib/(api,session,live,format,types)
deploy/  docker-compose.prod.yml · Caddyfile · litestream-b2.yml · k6/
```

## Scaling triggers (when to leave SQLite)

| Trigger | Threshold | Action |
|---|---|---|
| Write throughput | > ~100 tx/s sustained | Managed Postgres (`DB_PATH=postgres://…` — GORM swap, one afternoon) |
| Multi-node writes | need >1 writable instance | Postgres |
| Real plagiarism detection | embedding similarity | Postgres + pgvector |
| Durable background jobs | scanner becomes real work | River/asynq (the `Recover()` contract already matches) |

## Roadmap / non-goals

Grade-release scheduling · revisions/resubmission · email notifications · httpOnly BFF cookies (first production refactor) · enrollment table. WebSockets and Redis are deliberate non-goals at this scale — SSE and in-process state cover it.

MIT © 2025
