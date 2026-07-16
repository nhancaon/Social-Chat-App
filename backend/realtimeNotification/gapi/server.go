package gapi

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"realTimeNotification/models"
	pb "realTimeNotification/protos"

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
		// User has no active WebSocket connection right now.
		// The notification is still saved by the caller (e.g. the main API),
		// so it will show up next time the user fetches their notification list.
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

	// NOTE: This is the only goroutine that ever writes to a given
	// connection (the WebSocket handler in realtime/ only reads),
	// so this WriteJSON call is safe without an extra per-connection lock.
	if err := conn.WriteJSON(notification); err != nil {
		log.Printf("Error sending notification to websocket client %s: %v", req.MainUid, err)
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
