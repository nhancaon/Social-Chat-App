package realtime

import (
	"Server/database"
	"Server/models"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetUserFriends returns userID's mutual follows — the users present in both
// Followers and Following — since those are the only users whose online
// status this account should broadcast to and receive updates from.
func GetUserFriends(userID string) <-chan []string {
	result := make(chan []string, 1)

	go func() {
		defer close(result)

		objID, err := primitive.ObjectIDFromHex(userID)
		if err != nil {
			result <- nil
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var user models.UserModel
		if err := database.DB.Collection("users").FindOne(ctx, bson.M{"_id": objID}).Decode(&user); err != nil {
			result <- nil
			return
		}

		following := make(map[string]bool, len(user.Following))
		for _, id := range user.Following {
			following[id] = true
		}

		friends := make([]string, 0, len(user.Followers))
		for _, id := range user.Followers {
			if following[id] {
				friends = append(friends, id)
			}
		}

		result <- friends
	}()

	return result
}
