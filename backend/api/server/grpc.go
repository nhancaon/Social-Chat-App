package server

import (
	"Server/gapi"
	pb "SocialChatProtos"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func NewGRPCServer(port string) (*grpc.Server, net.Listener, error) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen on gRPC port %s: %w", port, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterRealtimeChatServiceServer(grpcServer, &gapi.Server{})
	reflection.Register(grpcServer)

	return grpcServer, lis, nil
}
