package gapi

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"Server/models"
	pb "SocialChatProtos"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.NotificationServiceClient
}

func NewClient() (*Client, error) {
	// conn, err := grpc.NewClient(":8090", grpc.WithTransportCredentials(insecure.NewCredentials()))

	//Devops docker compose usage
	conn, err := grpc.NewClient("SocialChatRealtimeNotification:8090", grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		return nil, fmt.Errorf("failed to connect to notification service: %w", err)
	}
	return &Client{
		conn:   conn,
		client: pb.NewNotificationServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) SendNotification(ctx context.Context, id, details, mainUID, targetID string,
	isRead bool, createdAt time.Time, userName, userAvatar string) error {

	request := &pb.NotificationRequest{
		Id:        id,
		Details:   details,
		MainUid:   mainUID,
		TargetId:  targetID,
		IsRead:    isRead,
		CreatedAt: timestamppb.New(createdAt),
		User: &pb.User{
			Name:   userName,
			Avatar: userAvatar,
		},
	}

	if _, err := c.client.SendNotification(ctx, request); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	return nil
}

// ---- Shared client (dial once, reuse for every notification) ----

var (
	sharedClientOnce sync.Once
	sharedClient     *Client
	sharedClientErr  error
)

func getSharedClient() (*Client, error) {
	sharedClientOnce.Do(func() {
		sharedClient, sharedClientErr = NewClient()
	})
	return sharedClient, sharedClientErr
}

// CloseSharedClient closes the shared connection. Call this once, on
// application shutdown (e.g. alongside your other graceful-shutdown steps).
func CloseSharedClient() error {
	if sharedClient != nil {
		return sharedClient.Close()
	}
	return nil
}

// SendNotification is the package-level entry point used by controllers.
// It reuses a single long-lived gRPC connection instead of dialing a new
// one on every call.
func SendNotification(notification models.Notification) error {
	client, err := getSharedClient()
	if err != nil {
		log.Printf("failed to get grpc client: %v", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.SendNotification(ctx,
		notification.ID.Hex(),
		notification.Details,
		notification.MainUID,
		notification.TargetID,
		notification.IsRead,
		notification.CreatedAt,
		notification.User.Name,
		notification.User.Avatar)
	if err != nil {
		log.Printf("failed to send notification: %v", err)
		return err
	}
	return nil
}
