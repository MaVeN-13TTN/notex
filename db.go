package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var notesCollection *mongo.Collection

// ConnectDB initializes the MongoDB connection
func ConnectDB() {
	// Load environment variables from .env file - provide absolute path
	envPath := ".env"
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("Error loading .env file from %s: %v", envPath, err)
		// Try alternate location
		log.Println("Trying to load from absolute path...")
		// Get current working directory
		cwd, err := os.Getwd()
		if err == nil {
			log.Printf("Current working directory: %s", cwd)
		}
	} else {
		log.Printf("Successfully loaded env file from: %s", envPath)
	}

	mongoURI := os.Getenv("MONGODB_URI")
	dbName := os.Getenv("MONGODB_DATABASE")
	log.Printf("MONGODB_URI: %s, MONGODB_DATABASE: %s", mongoURI, dbName)
	if mongoURI == "" || dbName == "" {
		log.Fatal("MONGODB_URI and MONGODB_DATABASE environment variables must be set")
	}

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(mongoURI).SetServerAPIOptions(serverAPI)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	client, err = mongo.Connect(ctx, opts)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Check the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	log.Println("Successfully connected to MongoDB!")
	notesCollection = client.Database(dbName).Collection("notes")
}

// DisconnectDB closes the MongoDB connection
func DisconnectDB() {
	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		} else {
			log.Println("Disconnected from MongoDB.")
		}
	}
}

// --- Database Operations ---

func CreateNote(note Note) (primitive.ObjectID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	note.CreatedAt = time.Now() // Set creation timestamp
	result, err := notesCollection.InsertOne(ctx, note)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return result.InsertedID.(primitive.ObjectID), nil
}

func GetAllNotes() ([]Note, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var notes []Note
	// Sort by creation date, newest first
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := notesCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &notes); err != nil {
		return nil, err
	}
	return notes, nil
}

func GetNoteByID(idHex string) (Note, error) {
	var note Note
	objectID, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return note, err // Invalid ID format
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = notesCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&note)
	return note, err // err will be mongo.ErrNoDocuments if not found
}

func DeleteNoteByID(idHex string) error {
	objectID, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		return err // Invalid ID format
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := notesCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err == nil && result.DeletedCount == 0 {
		return mongo.ErrNoDocuments // Indicate that the note wasn't found
	}
	return err
}
