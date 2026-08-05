package gapi

import (
	"Server/database"
	"Server/models"
	pb "SocialChatProtos"
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Server struct {
	pb.UnimplementedRealtimeChatServiceServer
}

func (s *Server) GetUserFollowingFollowers(ctx context.Context, req *pb.UserID) (*pb.UsersIDsListResponse, error) {
	UserSchema := database.DB.Collection("users")
	userID := req.GetUserid()

	if userID == "" {
		return nil, fmt.Errorf("user id is required")
	}

	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	var user models.UserModel
	err = UserSchema.FindOne(ctx, bson.M{"_id": uid}).Decode(&user)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	friendsMap := make(map[string]bool)
	for _, id := range user.Following {
		friendsMap[id] = true
	}
	for _, id := range user.Followers {
		friendsMap[id] = true
	}

	var friends []string
	for id := range friendsMap {
		friends = append(friends, id)
	}

	return &pb.UsersIDsListResponse{
		UserIdsLists: []*pb.UserIDsList{
			{UserIdsList: friends},
		},
	}, nil
}

func (s *Server) SendMessage(ctx context.Context, req *pb.MessageRequest) (*pb.MessageResponse, error) {
	if req.GetSender() == "" || req.GetReceiver() == "" {
		return nil, fmt.Errorf("sender and receiver are required")
	}

	UserSchema := database.DB.Collection("users")

	senderID, err := primitive.ObjectIDFromHex(req.GetSender())
	if err != nil {
		return nil, fmt.Errorf("invalid sender id: %w", err)
	}
	receiverID, err := primitive.ObjectIDFromHex(req.GetReceiver())
	if err != nil {
		return nil, fmt.Errorf("invalid receiver id: %w", err)
	}

	var sender, receiver models.UserModel
	err = UserSchema.FindOne(ctx, bson.M{"_id": senderID}).Decode(&sender)
	if err != nil {
		return nil, fmt.Errorf("sender not found")
	}
	err = UserSchema.FindOne(ctx, bson.M{"_id": receiverID}).Decode(&receiver)
	if err != nil {
		return nil, fmt.Errorf("receiver not found")
	}

	message := models.Message{
		Content:  req.GetContent(),
		Sender:   req.GetSender(),
		Receiver: req.GetReceiver(),
	}

	_, err = database.DB.Collection("messages").InsertOne(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("failed to save message to db")
	}

	unreadedmessagesSchema := database.DB.Collection("unReadmessages")

	filter := bson.M{"mainUserid": req.GetReceiver(), "otherUserid": req.GetSender()}
	update := bson.M{
		"$inc": bson.M{"numOfUnreadMessages": 1},
		"$set": bson.M{"isReadMessage": false},
	}
	opts := options.Update().SetUpsert(true)

	_, err = unreadedmessagesSchema.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to update unread messages: %w", err)
	}

	return &pb.MessageResponse{
		Message: req.GetContent(),
	}, nil
}
