package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	Name   string `json:"name" bson:"name"`
	Avatar string `json:"avatar,omitempty" bson:"avatar,omitempty"`
}

// interfaces
type Notification struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Details   string             `json:"details" bson:"details"`
	MainUID   string             `json:"mainuid" bson:"mainuid"`
	TargetID  string             `json:"targetid" bson:"targetid"`
	IsRead    bool               `json:"isRead" bson:"isRead"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	User      User               `json:"user" bson:"user"`
}
