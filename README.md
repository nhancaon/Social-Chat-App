# Social Chat App

[![Build](https://img.shields.io/badge/build-unknown-lightgrey)]() [![Version](https://img.shields.io/badge/version-0.1.0-blue)]() [![License](https://img.shields.io/badge/license-please%20specify-red)]()

Social Chat App is a lightweight social networking platform built with Go, Vue 3, MongoDB, and Docker. It supports user authentication, post creation, likes/comments, follow relationships, real-time chat, and notification delivery using both WebSocket and gRPC.

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

- User authentication with JWT-based sign in/sign up
- Social post flow: create, view, update, delete, like, and comment
- User profile and follow/unfollow actions
- Search for users and posts
- Real-time chat between users via WebSocket and gRPC
- Real-time notifications for likes, comments, and follow events
- REST API with Swagger documentation
- Dockerized setup for quick local deployment

## 🛠 Tech Stack

| Layer | Technology |
| --- | --- |
| Frontend | Vue 3, Quasar, Vue Router, Vuex, Axios |
| Backend | Go, Fiber v2, JWT, Swagger, gRPC |
| Real-time | WebSocket, gRPC |
| Database | MongoDB |
| Testing | Cypress, Go test, Testify |
| DevOps | Docker, Docker Compose |

## 📁 Project Structure

```text
backend/
  api/                  # Main REST API service written in Go
  realtimeChat/         # Real-time chat service
  realtimeNotification/ # Real-time notification service
frontend/              # Vue 3 frontend application
mongo.env.example      # MongoDB environment variables for Docker
docker-compose.yml     # Container orchestration
```

## 🚀 Getting Started

### Prerequisites

- Go 1.25+
- Node.js 18+
- Docker and Docker Compose
- MongoDB (if running locally)

### Installation

1. Clone the repository

```bash
git clone <your-repo-url>
cd Social-Chat-App
```

2. Start the backend

```bash
cd backend/api
go mod download
go run main.go
```

3. Start the frontend

```bash
cd frontend
npm install
npm run serve
```

### Environment Variables

Create a `.env` file in the backend API directory with the following values:

```env
PORT=5000
GRPC_PORT=5001
MONGODB_URI=mongodb://localhost:27017
JWT_SECRET=your-secret-key
```

If you are using Docker Compose, the MongoDB credentials are defined in `mongo.env.example`.

### Running the Project

#### Option 1: Docker Compose

```bash
docker compose up --build -d
```

Then open:

- Frontend: http://localhost
- Backend API: http://localhost:5000
- Swagger UI: http://localhost:5000/swagger/index.html

#### Option 2: Local development

Run the backend and frontend separately as shown in the installation steps above.

## 📖 API Documentation

The backend exposes REST endpoints for authentication, users, posts, chat, and notifications. Swagger documentation is available at:

- http://localhost:5000/swagger/index.html

### Main Endpoints

| Method | Endpoint | Description |
| --- | --- | --- |
| POST | `/user/signup` | Register a new user |
| POST | `/user/signin` | Sign in and receive a JWT |
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
| GET | `/notification/:userid` | Get notifications for a user |

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

This project includes Docker support for a simple deployment workflow.

```bash
docker compose up --build -d
```

For production, update secrets and credentials such as `JWT_SECRET` and MongoDB authentication values before deployment.

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

- [nhan](caonhan.work@example.com)
