package main

import(
    "go.mongodb.org/mongo-driver/mongo"
    "github.com/mongodb/mongo-go-driver/bson"
    "go.mongodb.org/mongo-driver/mongo/options"
    "context"
    "time"
    "log"
)

var(
    db *mongo.Database
    usersCollection *mongo.Collection
)

// Establish a connection with the MongoDB database
func init() {
    client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost"))
    if err != nil {
        log.Println(err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 20 * time.Second)
    defer cancel()

    if err = client.Connect(ctx); err != nil {
        log.Println(err)
    }

    db = client.Database("bookshelf")
    usersCollection = db.Collection("users")
   
    res, err := usersCollection.Indexes().CreateOne(
        context.Background(),
        mongo.IndexModel {
            Keys: bson.D{{"username", 1}},
            Options: options.Index().SetUnique(true),
        },
        options.CreateIndexes().SetMaxTime(20*time.Second),
    )
    if err != nil {
        log.Println(res, err)
    }
}

