.PHONY: proto

# Single source of truth for gRPC contracts shared by backend/api,
# backend/realtimeChat and backend/realtimeNotification (each references
# this module via "replace SocialChatProtos => ../protos" in its go.mod).
proto:
	protoc --go_out=. --go-grpc_out=. *.proto
