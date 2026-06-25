package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	Name    string `json:"name" bson:"name"`
	Avatart string `json:"avatart,omitempty" bson:"avatart,omitempty"`
}

// interfaces
type Notification struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Deatils   string             `json:"deatils" bson:"deatils"`
	MainUID   string             `json:"mainuid" bson:"mainuid"`
	TargetID  string             `json:"targetid" bson:"targetid"`
	IsReaded  bool               `json:"isreded" bson:"isreded"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	User      User               `json:"user" bson:"user"`
}
