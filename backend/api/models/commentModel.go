package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Comment struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	PostID    primitive.ObjectID `json:"postId,omitempty" bson:"postId,omitempty"`
	UserID    primitive.ObjectID `json:"userId,omitempty" bson:"userId,omitempty"`
	Value     string             `json:"value" bson:"value"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
}
