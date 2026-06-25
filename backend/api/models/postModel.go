package models

import (
    "time"

    "go.mongodb.org/mongo-driver/bson/primitive"
)

type PostModel struct {
    ID           primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
    Creator      string             `json:"creator" bson:"creator"`
    Title        string             `json:"title" bson:"title"`
    Message      string             `json:"message" bson:"message"`
    Name         string             `json:"name" bson:"name"`
    SelectedFile string             `json:"selectedFile" bson:"selectedFile"`
    Likes        []string           `json:"likes" bson:"likes"`
    Comments     []string           `json:"comments" bson:"comments"`
    CreatedAt    time.Time          `json:"createdAt" bson:"createdAt"`
}

// interfaces
type CreateOrUpdatePost struct {
    Title        string `json:"title" bson:"title" validate:"required"`
    Message      string `json:"message" bson:"message" validate:"required,min=5"`
    SelectedFile string `json:"selectedFile" bson:"selectedFile"`
}

// interfaces
type ComnmentPost struct {
    Value string `json:"value" bson:"value" validate:"required"`
}
