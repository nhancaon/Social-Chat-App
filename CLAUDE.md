# CLAUDE.md

Context for AI coding agents working in this repo. Read this before making
changes — it captures conventions and gotchas that aren't obvious from the
code alone, several of them learned the hard way during development.

## Project shape

- `backend/api/` — Go 1.25 + Fiber v2, module name `Server`. MongoDB, Redis,
  Kafka, S3.
- `frontend/` — Vue 3 + Quasar, Vuex 4, Options API.
- `terraform/` — root module provisions EKS + VPC + S3 (file storage);
  `terraform/rancher-host/` is a separate module for a persistent Rancher
  EC2 instance. Both use a shared S3 backend with native locking.
- `k8s/manifests/` — deployed via ArgoCD (`Automated` + `selfHeal`), watching
  this exact path. Don't put ArgoCD's own install manifest inside it —
  ArgoCD will try to redeploy itself.

## Backend conventions (`backend/api`)

- **Controllers**: no package-level collection vars. Each handler declares
  `var XSchema = database.DB.Collection("name")` locally, then
  `ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)`.
- **Auth**: `AuthMiddleware` sets `c.Locals("userId", userID)` after JWT
  validation. Every protected handler reads it via
  `c.Locals("userId").(string)` with an `ok` check → 401 if missing.
- **IDs**: convert via `primitive.ObjectIDFromHex`, 400 on failure.
- **Responses**: always `c.Status(fiber.StatusX).JSON(fiber.Map{...})`.
  Keys are inconsistent across handlers (`"error"`, `"message"`, `"data"`) —
  match whatever the neighboring handler in the same file already does.
- **Models** (`models/` package): `...Model` suffix for persisted documents,
  plain names for DTOs (e.g. `CreateOrUpdatePost`). ID field is always
  `primitive.ObjectID` tagged `json:"_id,omitempty" bson:"_id,omitempty"`.
  json tag = bson tag = camelCase, matching JS/Mongo convention. Sensitive
  fields get `json:"-"`.
- **Validation**: go-playground/validator, `validate:"required"` struct
  tags, applied via a middleware in `validation/` (see `ValidatePost`) —
  not inline in the handler.
- **Routes**: one file per feature in `routes/`, registered in
  `server/http.go`'s `NewHTTPServer()`. Don't wrap a route group in
  `app.Group(prefix, middleware)` if a sub-path under that prefix needs
  *different* middleware (e.g. `/chat/ws` needs `WSAuthMiddleware`, not
  `AuthMiddleware`) — Fiber's group middleware matches by prefix and will
  intercept it. Register routes individually instead (see `chat_routes.go`).
- **Background jobs** (`jobs/` package): one-shot mode via `main.go`'s
  `-job=<name>` flag, dispatched through `jobs.Run()`. Each job function
  takes a `context.Context`, does its Mongo/S3 work, returns an `error`; no
  HTTP server or Kafka hub gets started for job mode. Thresholds (retention
  hours, etc.) are env-var-configurable with a sane hardcoded default —
  follow this pattern for new jobs rather than hardcoding a constant.
- **Kafka**: each real-time feature (chat, notifications, user status,
  heartbeat) gets its own topic *and* every backend pod registers its own
  **per-node consumer group** (`<feature>-group-<nodeID>`), not a shared
  one. This is deliberate fan-out, not a bug — a shared group would load-balance
  messages across pods, but each pod needs to see every message to check
  whether it's holding the target user's WebSocket connection locally. New
  Kafka consumers should follow this same per-node-group pattern.
- **S3** (`storage/s3.go`): `S3Client`/`PresignClient` are package vars set
  by `InitS3()`. The S3 client is created with
  `RequestChecksumCalculation`/`ResponseChecksumValidation` forced to
  `WhenRequired` — aws-sdk-go-v2's newer default adds checksum headers to
  the signed request that a plain browser `fetch`/`axios` PUT/GET can never
  replicate, which breaks every presigned URL with `SignatureDoesNotMatch`
  if left at the default.
- **`database.Connect()`** logs and returns (no panic/exit) on failure —
  `database.DB` can be `nil` if Mongo isn't reachable at startup. Don't
  assume it's always non-nil.

## Frontend conventions (`frontend/src`)

- **Options API only** — no `<script setup>`, no Composition API anywhere
  in this codebase. Match the existing style.
- **Vuex 4**, namespaced modules under `store/`. Each module follows:
  `initialState()` → `getters` → `mutations` → `actions`, plus a
  `RESET_STATE` mutation (required — `store/plugins/resetOnLogout.js` warns
  at dev time if a module is missing one, and its state survives logout).
- **API layer**: one shared axios instance in `src/api/index.js` with a
  request interceptor that attaches `Authorization: Bearer <token>` from
  `localStorage.profile.token`, and `baseURL` from `src/runtime-config.js`
  (`window.APP_CONFIG?.API_URL || process.env.VUE_APP_API_URL || fallback`).
  Endpoint functions are grouped by feature with a comment header at the
  bottom of that one file — there's no per-feature api file.
- **Layering**: `component → Vuex action → api/index.js function`. Don't
  import `api/index.js` directly from a component.
- **Exception**: a request to a *different* origin (e.g. an S3 presigned
  URL) must use a bare `axios` import, not the shared `API` instance — the
  shared instance would incorrectly attach the JWT and prefix `baseURL`
  onto a third-party URL. See `MyFiles.vue`'s upload handler.
- **UI text is English**, even though commit messages and code comments in
  this repo are often Vietnamese.
- **`public/config.js`** gets rewritten by `docker-entrypoint.sh` at
  container start using the real `API_URL` env var. Keep the committed
  file's `API_URL` value falsy (`""`) — Vue CLI copies `public/` verbatim
  into `dist/`, so a truthy hardcoded value here always wins over
  `VUE_APP_API_URL` and silently breaks every non-Docker deployment (this
  broke the Vercel deploy once already).
- **Visual style**: `q-card flat bordered` with `border-radius: 12px`,
  subtle alternating row backgrounds (`rgba(0,0,0,0.02)`), rounded buttons.
  Match this rather than Quasar's defaults.

## Infra gotchas (all hit for real during development)

- **ArgoCD self-heal fights manual `kubectl` changes.** `chat-app`'s sync
  policy is `Automated` + `selfHeal`, and it watches the live cluster
  directly (not just polling Git every few minutes) — any `kubectl set env`
  or `kubectl edit` on a Git-tracked resource gets reverted almost
  instantly. To test a live env var change, pause it first:
  `argocd app set chat-app --sync-policy none`, make the change, then
  restore with `argocd app set chat-app --sync-policy automated --self-heal --auto-prune`
  when done. Forgetting the restore step leaves future commits unprotected
  from drift.
- **Single-broker Kafka needs an explicit replication factor.** Kafka's own
  default `offsets.topic.replication.factor` is 3; with one broker,
  `__consumer_offsets` silently fails to ever get created, and every
  consumer group permanently errors with "Group Coordinator Not Available"
  — not a transient error, it never recovers on its own.
  `KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1` fixes it (see
  `k8s/manifests/kafka.yaml`). App-level topics already set
  `ReplicationFactor: 1` explicitly in `kafka/manager.go`; this was the one
  internal topic still using the broker default.
- **The `backend` Service's LoadBalancer is a Classic ELB**, not an NLB —
  no `aws-load-balancer-type` annotation is set, so EKS's in-tree cloud
  provider defaults to Classic. Its default 60s idle timeout silently kills
  WebSocket connections left open without traffic (a backgrounded browser
  tab), which surfaces to the client as an HTTP 520 through Cloudflare on
  reconnect. Bumped to 3600s via the
  `service.beta.kubernetes.io/aws-load-balancer-connection-idle-timeout`
  annotation.
- **S3 CORS must list every origin actually in use**, not just the intended
  one — this app is served from both a custom domain and Vercel's own
  generated domain simultaneously; forgetting either one causes uploads
  from that origin to fail with a CORS preflight error, not an auth error.
- **Deleting a Service before tearing down the cluster can backfire.** If
  `chat-app` is still `Automated`+`selfHeal` when you delete the `backend`
  Service to avoid an orphaned ELB before `terraform destroy`, ArgoCD
  recreates it immediately (Git still declares it), provisioning a brand
  new ELB right as the cluster is being destroyed — which then really is
  orphaned, with no running cluster left to clean it up. Pause ArgoCD sync
  (see above) before deleting anything Git-tracked as part of teardown.
- **CI (`.github/workflows/`)**: `ci-cd.yml` chains
  `backend-tests` → `integration-frotend-tests` → `deploy-to-dockerhub`. A
  failing Cypress test silently blocks every downstream Docker image
  rebuild — check this chain first if "the deployed image seems stale."
  `integration-test.yml` stands up real Mongo/Redis/Kafka via Docker
  service containers (not mocks) and runs the real backend + frontend dev
  server against them before Cypress runs.
- **Local dev/testing stack** (mirrors CI, doesn't need AWS/EKS at all):
  ```
  docker run -d -p 27017:27017 mongo:8.0
  docker run -d -p 6379:6379 redis:7-alpine
  docker run -d -p 29092:29092 \
    -e KAFKA_NODE_ID=1 -e KAFKA_PROCESS_ROLES=broker,controller \
    -e KAFKA_LISTENERS="PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093,PLAINTEXT_HOST://0.0.0.0:29092" \
    -e KAFKA_ADVERTISED_LISTENERS="PLAINTEXT://localhost:9092,PLAINTEXT_HOST://localhost:29092" \
    -e KAFKA_CONTROLLER_QUORUM_VOTERS="1@localhost:9093" \
    -e KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER \
    -e KAFKA_LISTENER_SECURITY_PROTOCOL_MAP="PLAINTEXT:PLAINTEXT,CONTROLLER:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT" \
    -e KAFKA_INTER_BROKER_LISTENER_NAME=PLAINTEXT -e CLUSTER_ID=5L6g3nShT-eMctk--x86sw \
    apache/kafka:3.7.0
  cd backend/api && MONGODB_URI=mongodb://localhost:27017 REDIS_ADDR=localhost:6379 go run main.go --kafka=127.0.0.1:29092
  cd frontend && VUE_APP_API_URL=http://localhost:5000 npm run serve
  ```
