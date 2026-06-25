package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type UserModel struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Name      string             `json:"name" bson:"name"`
	Email     string             `json:"email" bson:"email" validate:"required"`
	Password  string             `json:"password,omitempty" bson:"password" validate:"required,min=5"`
	ImageUrl  string             `json:"imageUrl" bson:"imageUrl"`
	Bio       string             `json:"bio" bson:"bio"`
	Followers []string           `json:"followers" bson:"followers"`
	Following []string           `json:"following" bson:"following"`
}

// interfaces
type CreateUser struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

type LoginUser struct {
	Email    string
	Password string
}

type UpdateUser struct {
	Name     string
	ImageUrl string
	Bio      string
}
