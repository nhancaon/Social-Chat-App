# Social Chat App

[![Build](https://img.shields.io/badge/build-unknown-lightgrey)]() [![Version](https://img.shields.io/badge/version-0.1.0-blue)]() [![License](https://img.shields.io/badge/license-please%20specify-red)]()

Social Chat App is a lightweight social networking platform built with Go, Vue 3, MongoDB, Redis, and Kafka. It supports user authentication, post creation, likes/comments, follow relationships, and real-time chat/notifications delivered over WebSocket, fanned out across nodes via Kafka.

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
| Testing | Cypress, Go test, Testify |
| DevOps | Docker, Docker Compose, Terraform, Kubernetes (Helm, Rancher, ArgoCD) |

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

## 📁 Project Structure

```text
backend/
  api/                   # The backend service — REST API, WebSocket chat/notifications, Kafka fan-out
    kafka/                # Kafka producer/consumer, notification wire types, node heartbeat/leader election
    realtime/              # ChatHub/NotificationHub — WebSocket connection management per node
    controllers/            # HTTP handlers
    routes/                  # Route registration
    middleware/               # Auth (HTTP + WebSocket) middleware
    validation/                 # Request body validation
    database/                    # MongoDB + Redis clients
  docker-compose.yml      # Kafka (KRaft, multi-broker) + Redis, for native `go run main.go` local dev
frontend/                # Vue 3 frontend application
terraform/              # Provisions the AWS EKS lab cluster + Rancher host (see terraform/03-aws-eks.md)
k8s/                    # Helm values + manifests + step-by-step guides for Rancher/ArgoCD/EKS deployment
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
- **AWS EKS lab** — a cost-optimized, spin-up/tear-down Kubernetes deployment managed with Rancher (persistent EC2 host) + ArgoCD (local), self-hosted Kafka, and Redis (self-hosted or Upstash). Cluster provisioning is Terraform-based. Start at `k8s/01-local-management.md`, then `terraform/02-rancher-host.md`, then `terraform/03-aws-eks.md`; `k8s/04-connect-and-deploy.md` covers wiring them all together.

## 🤝 Contributing

Contributions are welcome.

1. Fork the repository
2. Create a new feature branch
3. Commit your changes
4. Open a pull request

## 📄 License

This project is currently not licensed. Please specify a license type in the badge and repository settings.

## 📧 Contact

For questions or collaboration, please contact:

- caonhan.work@gmail.com
