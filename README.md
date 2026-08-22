# Social Chat App

[![Build](https://img.shields.io/badge/build-unknown-lightgrey)]() [![Version](https://img.shields.io/badge/version-0.1.0-blue)]() [![License](https://img.shields.io/badge/license-please%20specify-red)]()

Social Chat App is a full-stack social networking platform built with Go, Vue 3, MongoDB, Redis, Kafka, and AWS S3. It supports user authentication, post creation, likes/comments, follow relationships, real-time chat/notifications delivered over WebSocket and fanned out across nodes via Kafka, and a file storage feature with automatic cold-storage archival (S3 → Glacier) to cut long-term storage cost.

## Table of Contents

- [Social Chat App](#social-chat-app)
  - [Table of Contents](#table-of-contents)
  - [✨ Features](#-features)
  - [🛠 Tech Stack](#-tech-stack)
  - [📁 Project Structure](#-project-structure)
  - [🚀 Getting Started](#-getting-started)
    - [Prerequisites](#prerequisites)
    - [Installation](#installation)
    - [Environment Variables](#environment-variables)
    - [Running the Project](#running-the-project)
      - [Option 1: Docker Compose](#option-1-docker-compose)
      - [Option 2: Local development](#option-2-local-development)
  - [📖 API Documentation](#-api-documentation)
    - [Main Endpoints](#main-endpoints)
  - [🧪 Testing](#-testing)
    - [Backend](#backend)
    - [Frontend](#frontend)
  - [📦 Deployment](#-deployment)
  - [🤝 Contributing](#-contributing)
  - [📄 License](#-license)
  - [📧 Contact](#-contact)

## ✨ Features

- User authentication with JWT-based sign in/sign up (password hash never returned in any API response)
- Social post flow: create, view, update, delete, like, and comment
- User profile and follow/unfollow actions
- Search for users and posts
- Real-time chat between users over WebSocket, fanned out across backend nodes via Kafka
- Real-time notifications for likes, comments, and follow events, same Kafka fan-out
- Multi-node aware: node ID + heartbeat/leader-election over Kafka, so multiple backend instances stay in sync
- Redis-backed caching (works with either a self-hosted Redis or a TLS managed provider like Upstash)
- File storage with S3 presigned upload/download (the backend never proxies file bytes), automatic archival to Glacier after 90 days of inactivity, on-demand restore with a realtime "file ready" notification, per-user storage quota, and trash with a 30-day recovery window before permanent deletion
- REST API with Swagger documentation
- Dockerized setup for local deployment, plus a Terraform + Kubernetes path for an AWS EKS lab deployment

## 🛠 Tech Stack

| Layer | Technology |
| --- | --- |
| Frontend | Vue 3, Quasar, Vue Router, Vuex, Axios |
| Backend | Go, Fiber v2, JWT, Swagger |
| Real-time | WebSocket, Kafka (`segmentio/kafka-go`) for cross-node fan-out |
| Database | MongoDB |
| Cache | Redis (`go-redis/v9`, optional TLS for managed providers) |
| File storage | AWS S3 (presigned URLs) + S3 Glacier (archival/restore), `aws-sdk-go-v2` |
| Testing | Cypress, Go test, Testify |
| DevOps | Docker, Docker Compose, Terraform, Kubernetes (Helm, Rancher, ArgoCD), Kubernetes CronJobs for archival/trash/restore |

> **Why Kafka instead of gRPC:** chat/notifications used to be separate
> microservices (`realtimeChat`, `realtimeNotification`) talking to
> `backend/api` over gRPC — point-to-point calls to a specific service
> instance. That breaks down under horizontal scaling: a message from a
> client on backend Pod A has no way to reach a client connected to Pod B
> without each Pod knowing about and calling every other Pod directly. Both
> services were folded into `backend/api`, and gRPC was replaced with a
> Kafka fan-out: every node publishes chat/notification events to a shared
> topic, and every node's consumer reads the full topic and forwards
> anything meant for its own locally-connected WebSocket clients — no node
> needs to know the others exist. This is what lets the backend scale via
> the Kubernetes HPA (see `k8s/`) without any peer-discovery logic.

> **Why tiered file storage:** most files get accessed heavily right after
> upload, then almost never again — but leaving everything in "hot" S3
> Standard storage forever means paying full price per GB for data nobody
> reads anymore. A daily `file-archive-scan` CronJob moves files idle 90+
> days to S3 Glacier, cutting per-GB storage cost by ~80%. The archive
> direction leans on AWS's own Lifecycle Policy (declared in
> `terraform/s3.tf`) where possible; the restore direction genuinely needs
> custom orchestration, since S3 never pushes a "restore complete" event —
> a `file-restore-poll` CronJob polls for it and, once ready, publishes to
> the same Kafka `notifications` topic chat/likes/comments already use, so
> the "file ready" push reaches the user over the existing WebSocket path
> instead of a bolted-on delivery mechanism.

## 📁 Project Structure

```text
CLAUDE.md                # Context for AI coding agents working in this repo — conventions + real gotchas
backend/
  api/                   # The backend service — REST API, WebSocket chat/notifications, Kafka fan-out
    kafka/                # Kafka producer/consumer, notification wire types, node heartbeat/leader election
    realtime/              # ChatHub/NotificationHub — WebSocket connection management per node
    controllers/            # HTTP handlers
    routes/                  # Route registration
    middleware/               # Auth (HTTP + WebSocket) middleware
    validation/                 # Request body validation
    database/                    # MongoDB + Redis clients
    storage/                      # S3 client, presigned URL helpers, Glacier archive/restore calls
    jobs/                          # One-shot background jobs (-job=<name> flag): archive-scan,
                                    # trash-purge, restore-poll, abandoned-upload-purge — run as K8s CronJobs
  docker-compose.yml      # Kafka (KRaft, multi-broker) + Redis, for native `go run main.go` local dev
frontend/                # Vue 3 frontend application (My Files page: src/views/MyFiles.vue)
terraform/              # Provisions AWS EKS + VPC + S3 file-storage bucket + IAM (see terraform/03-aws-eks.md);
                        # terraform/rancher-host/ is a separate module for a persistent Rancher EC2 instance
k8s/                    # Helm values + manifests (incl. file-jobs.yaml CronJobs) + step-by-step guides for
                        # Rancher/ArgoCD/EKS deployment and teardown
docker-compose.yml      # MongoDB + Redis + backend + frontend containers, for a full local stack
mongo.env.example      # MongoDB environment variables for Docker
```

## 🚀 Getting Started

### Prerequisites

- Go 1.25+
- Node.js 18+
- Docker and Docker Compose
- MongoDB, Redis, and Kafka (if running the backend locally without Docker — see `backend/docker-compose.yml`)

### Installation

1. Clone the repository

```bash
git clone <your-repo-url>
cd Social-Chat-App
```

2. Start Kafka + Redis for local dev (native `go run`, not the full Docker stack)

```bash
cd backend
docker compose up -d
```

3. Start the backend

```bash
cd backend/api
go mod download
go run main.go --kafka=127.0.0.1:29092
```

4. Start the frontend

```bash
cd frontend
npm install
npm run serve
```

### Environment Variables

Create a `.env` file in `backend/api/` — see `backend/api/.env.example` for the full list:

```env
MONGODB_URI=mongodb://admin:changeme@mongodb:27017
PORT=5000
GRPC_PORT=5001
JWT_SECRET=changeme
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
# set to true when REDIS_ADDR points at a provider reached over TLS (e.g. Upstash)
REDIS_TLS=false
# file storage (S3) — credentials come from the default AWS credential chain
# (env vars / IAM role), not from this file. Only region + bucket are app config.
AWS_REGION=ap-southeast-1
AWS_S3_BUCKET=changeme
```

The Kafka broker address is passed as a CLI flag, not an env var: `go run main.go --kafka=<host:port>` (defaults to `127.0.0.1:29092`, matching `backend/docker-compose.yml`).

If you are using the root Docker Compose stack, MongoDB credentials are defined in `mongo.env.example`.

### Running the Project

#### Option 1: Docker Compose

```bash
docker compose up --build -d
```

> The root `docker-compose.yml` currently starts MongoDB, Redis, backend, and frontend — it does **not** include Kafka. For a fully containerized run, also bring up the Kafka services from `backend/docker-compose.yml`, or run the backend natively as in Option 2.

Then open:

- Frontend: http://localhost
- Backend API: http://localhost:5000
- Swagger UI: http://localhost:5000/swagger/index.html

#### Option 2: Local development

Run Kafka/Redis, backend, and frontend separately as shown in the installation steps above.

## 📖 API Documentation

The backend exposes REST endpoints for authentication, users, posts, chat, and notifications, plus WebSocket endpoints for real-time chat/notifications. Swagger documentation is available at:

- http://localhost:5000/swagger/index.html

### Main Endpoints

| Method | Endpoint | Description |
| --- | --- | --- |
| POST | `/user/signup` | Register a new user |
| POST | `/user/signin` | Sign in and receive a JWT |
| GET | `/user/refresh` | Refresh a JWT using the current Bearer token |
| GET | `/user/getUser/:id` | Get user details and their posts |
| PATCH | `/user/Update/:id` | Update user profile |
| PATCH | `/user/:id/following` | Follow or unfollow a user |
| POST | `/posts` | Create a new post |
| GET | `/posts` | Get posts for the current user feed |
| GET | `/posts/search` | Search posts and users |
| GET | `/posts/:id` | Get a specific post |
| PATCH | `/posts/:id` | Update a post |
| POST | `/posts/:id/commentPost` | Add a comment to a post |
| PATCH | `/posts/:id/likePost` | Like or unlike a post |
| DELETE | `/posts/:id` | Delete a post |
| POST | `/chat/sendmessage` | Send a chat message |
| GET | `/chat/getmsgsbynums` | Retrieve chat history |
| GET | `/chat/get-user-unreadmsg` | Get unread message count |
| GET | `/chat/ws` | WebSocket upgrade for real-time chat |
| GET | `/notification/:userid` | Get notifications for a user |
| GET | `/notification/mark-notification-asreaded` | Mark a notification as read |
| GET | `/notification/ws` | WebSocket upgrade for real-time notifications |
| POST | `/files/upload-url` | Request a presigned S3 upload URL (client uploads directly to S3) |
| POST | `/files/:id/confirm` | Confirm an upload finished, credit it against the user's quota |
| GET | `/files` | List the current user's files |
| GET | `/files/:id/download-url` | Request a presigned S3 download URL (409 if archived to Glacier) |
| POST | `/files/:id/restore` | Request restoring an archived (Glacier) file |
| DELETE | `/files/:id` | Move a file to trash (soft delete) |

## 🧪 Testing

### Backend

```bash
cd backend/api
make test
```

### Frontend

```bash
cd frontend
npm run lint
npm run cy:run:auth
```

## 📦 Deployment

Two paths, depending on the goal:

- **Docker Compose** — quick local run, see [Running the Project](#running-the-project) above. For production, update secrets/credentials such as `JWT_SECRET` and MongoDB/Redis authentication values before deploying.
- **AWS EKS lab** — a cost-optimized, spin-up/tear-down Kubernetes deployment managed with Rancher (persistent EC2 host) + ArgoCD (local), self-hosted Kafka, and Redis (self-hosted or Upstash), plus an S3 bucket for file storage. Cluster provisioning is Terraform-based. Start at `k8s/01-local-management.md`, then `terraform/02-rancher-host.md`, then `terraform/03-aws-eks.md`; `k8s/04-connect-and-deploy.md` covers wiring them all together (including the S3 bucket and file-storage CronJobs) and, at the end, how to tear everything down cleanly to stop billing.

## 🤝 Contributing

Contributions are welcome.

1. Fork the repository
2. Create a new feature branch
3. Commit your changes
4. Open a pull request

If you're working in this repo with an AI coding agent (Claude Code or
similar), read [`CLAUDE.md`](CLAUDE.md) first — it captures conventions and
real gotchas (Kafka replication factor, ArgoCD self-heal vs. manual
`kubectl` changes, presigned URL signing, etc.) so the agent doesn't have to
rediscover them.

## 📄 License

This project is currently not licensed. Please specify a license type in the badge and repository settings.

## 📧 Contact

For questions or collaboration, please contact:

- caonhan.work@gmail.com
