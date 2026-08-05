package gapi

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"realTimeNotification/models"
	pb "SocialChatProtos"

	"github.com/gofiber/websocket/v2"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// notificationServer implements the gRPC NotificationService.
// It receives notification requests from other backend services and
// forwards them to the matching WebSocket connection, if the target
// user currently has one open.
type notificationServer struct {
	pb.UnimplementedNotificationServiceServer
	wsMu *sync.Mutex
	ws   map[string]*websocket.Conn
}

func (s *notificationServer) SendNotification(ctx context.Context, req *pb.NotificationRequest) (*emptypb.Empty, error) {
	log.Printf("Sending notification to user %s: %s", req.MainUid, req.Details)

	s.wsMu.Lock()
	conn, connected := s.ws[req.MainUid]
	s.wsMu.Unlock()

	if !connected {
		log.Printf("[DEBUG] No active connection for user %s (current connected users: %v)", req.MainUid, s.ws)
		return &emptypb.Empty{}, nil
	}

	notification := models.Notification{
		ID:        req.Id,
		MainUID:   req.MainUid,
		Details:   req.Details,
		TargetID:  req.TargetId,
		IsRead:    req.IsRead,
		CreatedAt: req.CreatedAt.AsTime(),
		User: models.User{
			Name:   req.User.Name,
			Avatar: req.User.Avatar,
		},
	}

	log.Printf("[DEBUG] Found connection for %s, writing...", req.MainUid)
	if err := conn.WriteJSON(notification); err != nil {
		log.Printf("[DEBUG] WriteJSON failed: %v", err)
	} else {
		log.Printf("[DEBUG] WriteJSON succeeded")
	}

	return &emptypb.Empty{}, nil
}

// StartGRPCServer starts listening on :8090 and serving the notification
// gRPC service in the background. It returns the *grpc.Server so the
// caller can gracefully stop it on shutdown.
func StartGRPCServer(ws map[string]*websocket.Conn, wsMu *sync.Mutex) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", ":8090")
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port 8090: %w", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterNotificationServiceServer(grpcServer, &notificationServer{ws: ws, wsMu: wsMu})

	go func() {
		log.Println("gRPC server running on port 8090")
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()

	return grpcServer, nil
}
