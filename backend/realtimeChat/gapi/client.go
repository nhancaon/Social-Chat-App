package gapi

import (
	"context"
	"log"
	"sync"
	"time"

	"SocialChatProtos"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	grpcServerAddr = ":5001"
	rpcTimeout     = 5 * time.Second
)

var (
	sharedConn     *grpc.ClientConn
	sharedConnOnce sync.Once
	sharedConnErr  error
)

func getClient() (protos.RealtimeChatServiceClient, error) {
	sharedConnOnce.Do(func() {
		// sharedConn, sharedConnErr = grpc.NewClient(grpcServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))

		//Devops docker compose usage
		sharedConn, sharedConnErr = grpc.NewClient("SocialChatRealtimeChat:5001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	})
	if sharedConnErr != nil {
		return nil, sharedConnErr
	}
	return protos.NewRealtimeChatServiceClient(sharedConn), nil
}

func GetFollowingFollowersClient(id string) ([]*protos.UserIDsList, error) {
	client, err := getClient()
	if err != nil {
		log.Printf("Client: did not connect: %v", err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	req := &protos.UserID{Userid: id}
	resp, err := client.GetUserFollowingFollowers(ctx, req)
	if err != nil {
		log.Printf("GetFollowingFollowersClient: error calling GetUserFollowingFollowers grpc: %v", err)
		return nil, err
	}
	return resp.GetUserIdsLists(), nil
}

func SendMessageClient(sender, receiver, content string) error {
	client, err := getClient()
	if err != nil {
		log.Printf("Client: did not connect: %v", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	req := &protos.MessageRequest{
		Sender:   sender,
		Receiver: receiver,
		Content:  content,
	}
	_, err = client.SendMessage(ctx, req)
	if err != nil {
		log.Printf("SendMessageClient: error calling SendMessage grpc: %v", err)
		return err
	}
	return nil
}
