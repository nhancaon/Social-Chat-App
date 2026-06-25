package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Message struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Content   string             `json:"content" bson:"content"`
	Sender    string             `json:"sender" bson:"sender"`
	Receiver  string             `json:"receiver" bson:"receiver"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
}

// interfaces
type SendMessageM struct {
	Content  string `json:"content" bson:"content"  validate:"required,min=5"`
	Sender   string `json:"sender" bson:"sender"  validate:"required"`
	Receiver string `json:"receiver" bson:"receiver"  validate:"required"`
}
