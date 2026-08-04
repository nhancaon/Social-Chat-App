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
	ID       primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Details  string             `json:"details" bson:"details"`
	MainUID  string             `json:"mainUid" bson:"mainuid"`
	TargetID string             `json:"targetId" bson:"targetid"`
	UserID   string             `json:"userId" bson:"userid"`
	// User is a snapshot of the actor's name/avatar taken at creation time,
	// so the notification list can render without a join back to users.
	User      User      `json:"user" bson:"user"`
	IsRead    bool      `json:"isRead" bson:"isRead"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
}
