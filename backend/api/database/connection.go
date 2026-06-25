package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client
var DB *mongo.Database

func Connect() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mongoUri := "mongodb://localhost:27017"
	clientOptions := options.Client().ApplyURI(mongoUri)

	var err error
	Client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		fmt.Println("Failed to connect to database:", err.Error())
		return
	}

	if err = Client.Ping(ctx, nil); err != nil {
		fmt.Println("Failed to ping database:", err.Error())
		return
	}

	DB = Client.Database("social-chat")
	fmt.Println("Connected to MongoDB!")
}
