package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UnReadMsg struct {
	ID                  primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	MainUserid          string             `json:"mainUserid" bson:"mainUserid"`
	OtherUserid         string             `json:"otherUserid" bson:"otherUserid"`
	NumOfUnreadMessages int                `json:"numOfUnreadMessages" bson:"numOfUnreadMessages"`
	IsReadMessage       bool               `json:"isReadMessage" bson:"isReadMessage"`
}
