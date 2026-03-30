# WebSocket Live Screen Implementation Plan

## Problem Statement

Current live screen uses **HTTP polling** (mobile polls `GET /api/playlists/:id/live` every 1.5 seconds). This works but is inefficient for real-time sync. User wants to migrate to **WebSocket** for push-based updates when backend runs on **traditional hosting (not serverless)**.

## Current Architecture

### Backend (be-weworship)
- **Framework**: Go + Fiber (currently serverless-compatible via `@vercel/go`)
- **Live State Storage**: Redis with `LiveState` struct (song_index, scroll_ratio, leader_user_id)
- **Existing Endpoints**:
  - `POST /api/playlists/:id/live` — Start session
  - `PUT /api/playlists/:id/live/state` — Update state (leader posts updates)
  - `GET /api/playlists/:id/live` — Get state (followers poll this)
  - `DELETE /api/playlists/:id/live` — End session

### Mobile (mobile-weworship)
- **Current Implementation**: `app/playlist/[id]/present.tsx` polls every 1.5s
- **Service**: `services/liveSessionService.ts` wraps HTTP API calls
- **No WebSocket client** currently exists

## Proposed Solution: WebSocket Architecture

### High-Level Design

```
┌─────────────────────────────────────────────────────────────┐
│                      Mobile Clients                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Leader     │  │  Follower 1  │  │  Follower 2  │      │
│  │ (sends cmds) │  │ (receives)   │  │ (receives)   │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                  │                  │               │
│         └──────────────────┼──────────────────┘               │
│                            │ WebSocket                        │
└────────────────────────────┼──────────────────────────────────┘
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                    Go Backend Server                         │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │          WebSocket Connection Manager                   │ │
│  │  • Track active connections per playlist               │ │
│  │  • Handle join/leave/update messages                   │ │
│  │  • Broadcast state changes to room                     │ │
│  └────────────────────┬───────────────────────────────────┘ │
│                       │                                      │
│  ┌────────────────────▼───────────────────────────────────┐ │
│  │              Redis Pub/Sub                             │ │
│  │  • Channel: live:events:playlist:{id}                  │ │
│  │  • Persist state in live:playlist:{id} (existing)      │ │
│  │  • Publish updates to all server instances            │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Message Protocol

**Client → Server (JSON over WebSocket)**:
```typescript
// Join a live session
{ "type": "join", "playlist_id": 123 }

// Leader updates state
{ "type": "update", "playlist_id": 123, "song_index": 5, "scroll_ratio": 0.42 }

// Leave session
{ "type": "leave", "playlist_id": 123 }

// Heartbeat
{ "type": "ping" }
```

**Server → Client (JSON over WebSocket)**:
```typescript
// State update broadcast
{
  "type": "state_update",
  "playlist_id": 123,
  "song_index": 5,
  "scroll_ratio": 0.42,
  "leader_user_id": 456,
  "updated_at": "2026-03-27T12:00:00Z"
}

// Session started
{ "type": "session_started", "playlist_id": 123, "leader_user_id": 456 }

// Session ended
{ "type": "session_ended", "playlist_id": 123 }

// Error
{ "type": "error", "message": "Not authorized" }

// Pong
{ "type": "pong" }
```

### Redis Strategy

**Existing Redis Keys** (keep these):
- `live:playlist:{id}` → LiveState JSON (song_index, scroll_ratio, leader_user_id, etc.)

**New Pub/Sub Channel**:
- `live:events:playlist:{id}` → Broadcast state changes to all backend instances (for horizontal scaling)

## Implementation Plan

### Phase 1: Backend WebSocket Infrastructure

#### 1. Add WebSocket Library
- Add `github.com/gorilla/websocket` to `go.mod`
- Create `api/platform/websocket_manager.go` — connection pool + room management

#### 2. WebSocket Connection Manager
**File**: `api/platform/websocket_manager.go`

Core responsibilities:
- Upgrade HTTP → WebSocket
- Track connections by playlist room
- Authenticate via JWT (reuse existing auth)
- Handle message routing (join, leave, update, ping/pong)
- Broadcast to room members
- Clean up on disconnect

Data structures:
```go
type WSManager struct {
  rooms map[int]*Room // playlist_id → Room
  mu    sync.RWMutex
  redis *redis.Client
}

type Room struct {
  PlaylistID int
  Conns      map[*WSConnection]bool
  mu         sync.RWMutex
}

type WSConnection struct {
  Conn      *websocket.Conn
  UserID    int
  Send      chan []byte
  Manager   *WSManager
  RoomID    int
}
```

#### 3. Update Live Service
**File**: `api/services/playlist_service.go`

- Keep existing Redis `StartLive`, `EndLive`, `UpdateLiveState`, `GetLiveState` methods
- Add Redis pub/sub publishing: when state changes, publish to `live:events:playlist:{id}`
- Add broadcast helper: `BroadcastLiveStateChange(playlistID int, state *LiveState)`

#### 4. WebSocket Handler
**File**: `api/handlers/websocket_handler.go`

- `UpgradeToWebSocket(c *fiber.Ctx)` → upgrades HTTP to WS, authenticates JWT
- Message dispatcher for `join`, `leave`, `update`, `ping`

#### 5. Register WebSocket Route
**File**: `api/handlers/router.go`

Add route:
```go
app.Get("/ws/live", middleware.WSAuth(), h.UpgradeToWebSocket)
```

#### 6. Redis Pub/Sub Listener
**File**: `api/platform/websocket_manager.go`

- Subscribe to `live:events:playlist:*` on startup
- When message received, broadcast to all connections in that room
- Handles multi-instance deployment (each server instance listens)

### Phase 2: Mobile WebSocket Client

#### 1. Install WebSocket Library
```bash
npm install react-native-fast-websocket
```
(or use built-in `WebSocket` API)

#### 2. Create WebSocket Service
**File**: `services/liveWebSocketService.ts`

```typescript
export class LiveWebSocketService {
  private ws: WebSocket | null = null;
  private listeners: Map<string, (data: any) => void> = new Map();
  
  connect(token: string): Promise<void>
  disconnect(): void
  
  joinLiveSession(playlistId: number): void
  leaveLiveSession(playlistId: number): void
  updateLiveState(playlistId: number, songIndex: number, scrollRatio: number): void
  
  onStateUpdate(callback: (state: LiveState) => void): () => void
  onSessionStarted(callback: (data: any) => void): () => void
  onSessionEnded(callback: (data: any) => void): () => void
}
```

Features:
- Auto-reconnect on disconnect (with exponential backoff)
- Heartbeat/ping every 30s
- Event emitter pattern for state updates
- JWT token in connection URL or initial message

#### 3. Update Presentation Screen
**File**: `app/playlist/[id]/present.tsx`

Changes:
- Remove polling logic (`useEffect` with `setInterval`)
- Replace with WebSocket subscription:
  ```typescript
  useEffect(() => {
    const unsubscribe = liveWebSocketService.onStateUpdate((state) => {
      // Update local state
      setCurrentSongIndex(state.song_index);
      setScrollRatio(state.scroll_ratio);
    });
    
    liveWebSocketService.joinLiveSession(playlistId);
    
    return () => {
      liveWebSocketService.leaveLiveSession(playlistId);
      unsubscribe();
    };
  }, [playlistId]);
  ```
- Leader sends updates via WebSocket instead of HTTP POST

### Phase 3: Backward Compatibility & Migration

#### 1. Keep HTTP Endpoints (Hybrid Mode)
- Do NOT remove existing polling endpoints
- Support both WebSocket AND HTTP polling during migration
- Mobile can fall back to polling if WebSocket fails

#### 2. Feature Flag
Add environment variable:
```
ENABLE_WEBSOCKET=true  # on traditional server
ENABLE_WEBSOCKET=false # on Vercel serverless
```

- If `false`, WebSocket endpoint returns 503
- Mobile detects this and falls back to polling

#### 3. Deployment Strategy
1. Deploy BE with WebSocket to traditional server (DigitalOcean, AWS ECS, etc.)
2. Test WebSocket with a single mobile client
3. Roll out mobile app update with WebSocket
4. Monitor for issues, fall back to polling if needed
5. After 2 weeks, deprecate HTTP polling endpoints (optional)

### Phase 4: Testing & Optimization

#### 1. Load Testing
- Test with 50+ concurrent connections per playlist
- Verify Redis pub/sub works across multiple server instances
- Check memory usage for connection pool

#### 2. Mobile Edge Cases
- Background → foreground (reconnect)
- Network switch (WiFi → cellular)
- Token expiry (refresh and reconnect)
- App killed (graceful disconnect)

#### 3. Monitoring
- Log connection count per room
- Track reconnect frequency
- Monitor Redis pub/sub latency
- Alert if > 100ms message delay

## Technical Decisions

### Why Gorilla WebSocket?
- Most popular Go WebSocket library (15k+ stars)
- Production-ready, well-tested
- Works with Fiber via HTTP upgrade
- Supports ping/pong for connection health

### Why Redis Pub/Sub?
- Already using Redis for live state storage
- Enables horizontal scaling (multi-instance deployment)
- Low latency (~1-5ms)
- Built-in pattern matching for channels

### Why Keep HTTP Endpoints?
- Graceful degradation (serverless fallback)
- Debugging (can test via Postman)
- Backward compatibility with old mobile clients
- Simpler monitoring (standard HTTP metrics)

### Authentication Strategy
- Reuse existing JWT middleware
- Parse token from query param: `wss://api/ws/live?token=<JWT>`
- Validate on connection upgrade
- Reject if expired or invalid

## Files to Create/Modify

### Backend (be-weworship)

**New Files**:
1. `api/platform/websocket_manager.go` — Connection pool, room management, pub/sub listener
2. `api/handlers/websocket_handler.go` — WebSocket upgrade, message dispatcher

**Modified Files**:
1. `go.mod` — Add `github.com/gorilla/websocket v1.5.1`
2. `api/handlers/router.go` — Register `/ws/live` route
3. `api/services/playlist_service.go` — Publish Redis pub/sub events on state changes
4. `api/platform/context.go` — Initialize WSManager singleton
5. `api/handlers/handler.go` — Add WSManager to Handler struct
6. `api/middleware/auth.go` — Add `WSAuth()` for WebSocket token validation (query param)

### Mobile (mobile-weworship)

**New Files**:
1. `services/liveWebSocketService.ts` — WebSocket client wrapper
2. `types/websocket.ts` — TypeScript types for WS messages

**Modified Files**:
1. `package.json` — Add `react-native-fast-websocket` (or use native WebSocket API)
2. `app/playlist/[id]/present.tsx` — Replace polling with WebSocket subscription
3. `services/liveSessionService.ts` — Keep HTTP fallback methods, add feature flag check
4. `constants/config.ts` — Add `WS_BASE_URL` (e.g., `wss://api.weworship.com`)

## Deployment Notes

### Traditional Server Requirements
- **Persistent process** (not serverless)
- **Long-lived TCP connections** support
- **Redis instance** with pub/sub enabled
- **Load balancer** with sticky sessions (optional, pub/sub handles multi-instance)

### Recommended Hosting Options
1. **DigitalOcean App Platform** — Go buildpack, easy scaling
2. **AWS ECS Fargate** — Container-based, auto-scaling
3. **Fly.io** — Edge deployment, built-in Redis
4. **Railway** — Simple deploy, managed Redis addon
5. **Self-hosted VPS** (DigitalOcean Droplet, Linode, etc.)

### Environment Variables to Add
```bash
REDIS_URL=redis://...          # Already exists
ENABLE_WEBSOCKET=true          # Feature flag
WS_PING_INTERVAL=30s           # Heartbeat interval
WS_WRITE_TIMEOUT=10s           # Message send timeout
WS_READ_TIMEOUT=60s            # Pong response timeout
WS_MAX_CONNECTIONS=5000        # Per-instance limit
```

## Success Metrics

- **Latency**: State updates propagate in < 100ms (vs 1.5s polling)
- **Bandwidth**: 90% reduction (no repeated GET requests)
- **Battery**: 20% less drain on mobile (no constant polling)
- **Scalability**: Support 100+ concurrent users per playlist
- **Reliability**: Auto-reconnect on disconnect within 5s

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| WebSocket disconnects frequently | Auto-reconnect with exponential backoff + HTTP fallback |
| Redis pub/sub fails | Each server maintains local connection pool; degraded but functional |
| Mobile battery drain from WS | Use heartbeat every 30s (not 1.5s polling) |
| Scaling across multiple servers | Redis pub/sub broadcasts to all instances |
| Token expiry during long session | Refresh token, reconnect with new token |
| Network switch (WiFi → LTE) | Detect via `NetInfo`, reconnect immediately |

## Future Enhancements (Out of Scope)

- Presence indicators (show who's online in a live session)
- Direct messages between team members during live
- Audio sync (play/pause YouTube/Spotify for everyone)
- Recording live sessions for playback
- Analytics on live session engagement

## Notes

- This plan assumes **traditional server deployment** (not Vercel serverless)
- HTTP polling endpoints remain for backward compatibility
- Redis pub/sub enables horizontal scaling (multiple server instances)
- Mobile app uses WebSocket with automatic fallback to HTTP
- Implementation is phased: BE first, then mobile, then optimize

---

# FCM Fix + Debug Logging — be-weworship

## Problem
1. `context.go` only initializes FCM if `FCM_CREDENTIALS_PATH` is set — Vercel uses `FCM_CREDENTIALS_JSON`, so FCM is always disabled on production
2. Goroutines in `notification_service.go` are killed by Vercel serverless after handler returns
3. No logs to diagnose FCM delivery

## Fix Plan
1. `api/platform/context.go` — also check `FCM_CREDENTIALS_JSON` in the init condition
2. `api/services/notification_service.go` — remove goroutines, run FCM synchronously with a 10s timeout context; add debug logs at each step
3. `api/providers/fcm.go` — add richer error logging (log full response on failure)

---

# Notification Inbox — be-weworship

## Goal
Complete the notification inbox API. `NotifyNewSong` is a broadcast (user_id = NULL, visible to all). Targeted notifications (playlist update, member left, song request) are per-user. Broadcasts never count toward the unread badge.

## Design: Broadcast vs Targeted
- `user_id = NULL` → broadcast (new song) — shown to all users, no per-user read tracking
- `user_id = N` → targeted — full is_read tracking, counts toward unread badge

## Files to change
1. `schema.sql` — relax `user_id NOT NULL` → `user_id INTEGER NULL`; fix indexes
2. `notification_repository.go` — nullable UserID, SaveBroadcastNotification, update ListByUserID/CountUnread
3. `notification_service.go` — persist to inbox in all Notify* methods; add GetNotifications/MarkAsRead/GetUnreadCount
4. `notification_handler.go` — add GetNotifications, MarkNotificationRead, GetUnreadCount handlers
5. `router.go` — register 3 new routes

## New Routes
| Method | Route | Description |
|--------|-------|-------------|
| GET | `/api/notifications` | Inbox: user's targeted + all broadcasts, paginated |
| GET | `/api/notifications/unread-count` | Count of unread targeted notifications only |
| POST | `/api/notifications/:id/read` | Mark a targeted notification as read |

---

# Song Request Feature — be-weworship

## Goal
Allow mobile users to request a song to be added. They submit a song title + reference link (YouTube/Spotify/Apple Music/etc). Admin can view and manage requests.

## Files to change
- `schema.sql` — new `song_requests` table
- `api/models/song.go` — `SongRequest` struct
- `api/repositories/song_repository.go` — CRUD methods
- `api/services/song_service.go` — business logic + validation
- `api/handlers/song_handler.go` — HTTP handlers
- `api/handlers/router.go` — route registration

## API Design
| Method | Route | Auth | Description |
|--------|-------|------|-------------|
| POST | `/api/songs/request` | auth required | User submits a song request |
| GET | `/api/song-requests` | auth required | User lists their own requests |
| DELETE | `/api/song-requests/:id` | auth required | User deletes own request (403 if not owner) |
| GET | `/api/admin/song-requests` | admin | List all requests (filter by status) |
| PUT | `/api/admin/song-requests/:id` | admin | Update status + admin notes |

## Data shape
**Request body** (mobile → BE):
```json
{ "song_title": "string (required)", "reference_link": "string (required)" }
```
**DB row**: id, user_id, song_title NOT NULL, reference_link NOT NULL, status (pending/approved/rejected), admin_notes, createdAt, updatedAt

---

# Token Efficiency Research — AI-Assisted Development

## Research Question
What factors most affect token efficiency when using GitHub Copilot CLI with `copilot resume --session-id` across a long-running project?

## Dataset
- Session: weworship fullstack (BE + mobile + admin)
- Duration: 15 days, 210 user messages, 2,832 tool calls
- 16 compactions, 15 resumes, 16 checkpoints

---

## Key Findings

### 1. PROMPT QUALITY (Impact: ~5% of waste)
- **Surprising**: Failure rate is ~6% regardless of prompt quality
- HIGH quality prompts use **40% fewer tools** (7.8 vs 12.8)
- Length doesn't matter — **specificity** does
- Sweet spot: medium length (80-200 chars) with error/screenshot context
- Worst: short+vague (30-80 chars, no context) → 9% still rate

### 2. TOOL SELECTION (Impact: ~10% of waste)
- Efficient turns: view:edit ratio = **0.43** (more edits than views)
- Inefficient turns: view:edit ratio = **0.73** (70% more views)
- Best tool: `edit` (2.7% fail) — most reliable
- Worst tool: `apply_patch` (25% fail) — most error-prone
- `bash` most used (1,226 calls) with 4.9% fail rate

### 3. REPEATED FILE READS (Impact: ~25% of waste — BIGGEST)
- song/[id].tsx read **41 times** across 16 different tasks
- router.go read **16 times** across 12 different tasks
- 145 same-file re-edits within single turns
- 24 edit→build-fail→re-edit cycles (~12K tokens)
- **This is the #1 efficiency killer**

### 4. CONTEXT vs MEMORY (Impact: ~8% of waste)
- Post-compaction turns use **+89% more tools** than normal
- Post-resume turns use **+40% more tools** than normal
- BUT: failure rate stays the SAME (~6%)
- **Memory preserves correctness; context loss only costs speed**
- Checkpoint summaries work well for "what happened"
- But AI still needs file reads for "what does the code look like now"

### 5. REWORK / 'STILL' FOLLOW-UPS (Impact: ~7% of waste)
- 13 "still" messages = user said AI's fix didn't work
- 38% (5/13) were convention violations (theme, mutations)
- 62% required real user feedback (screenshots, device testing)

---

## Ranked Efficiency Factors

| Rank | Factor | % of Token Budget | Preventable? | Fix |
|------|--------|------------------|--------------|-----|
| 1 | Repeated file reads | ~25% | YES | file-map + schema in context-keeper |
| 2 | Build/check cycles | ~10% | Partial | Batch edits, fewer tsc checks |
| 3 | Context recovery | ~8% | YES | Persistent memory entries |
| 4 | Rework (still) | ~7% | Partial | Conventions + user screenshots |
| 5 | Prompt quality | ~5% | YES | Specific prompts with context |
| - | Useful implementation | ~45% | - | (this is the goal) |

---

## Actionable Recommendations for Developer

### DO (high impact)
- [x] Use `copilot resume` (keeps checkpoint context — already doing this)
- [ ] Paste error output + screenshot together (highest quality prompts had 0% fail)
- [ ] For UI changes: always mention which screen + current behavior + expected
- [ ] Save context-keeper entries for: file layouts, API routes, type schemas

### DON'T (waste tokens)
- [ ] Don't send vague 1-line prompts without context for complex UI changes
- [ ] Don't ask for multiple unrelated changes in one message (causes 30+ tool turns)
- [ ] Don't repeat "still X" without adding NEW information (same prompt = same result)

### FOR CONTEXT-KEEPER (highest ROI entries to save)
1. **file-map**: mobile route structure + component responsibilities
2. **api-catalog**: BE endpoint list with auth/groups
3. **schema**: BESong→Song mapping, Playlist types
4. **convention**: useAppTheme() mandatory, no in-place mutations
5. **gotcha**: expo-file-system legacy import, SafeAreaView package
