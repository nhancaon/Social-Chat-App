# Stage 1: Build
FROM golang:1.26.4 AS builder
WORKDIR /usr/src/app

# Shared gRPC contracts (single source of truth, used by all backend services
# via a "replace SocialChatProtos => ../protos" in each service's go.mod)
COPY backend/protos ./backend/protos

COPY backend/api/go.mod backend/api/go.sum ./backend/api/
WORKDIR /usr/src/app/backend/api
RUN go mod download

WORKDIR /usr/src/app
COPY backend/api ./backend/api
WORKDIR /usr/src/app/backend/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /usr/src/app/main .

# Stage 2: Run (image nhẹ hơn, không cần toàn bộ toolchain Go)
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /usr/src/app/main .
# Không copy .env vào image — secret/config truyền lúc "docker run"
# qua -e hoặc docker-compose "environment:", không đóng gói cứng vào image

EXPOSE 5000
CMD [ "./main" ]