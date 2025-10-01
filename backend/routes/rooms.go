package routes

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"arguehub/db"
	"arguehub/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Room represents a debate room.
type Room struct {
	ID           string        `json:"id" bson:"_id"`
	Type         string        `json:"type" bson:"type"`
	Participants []Participant `json:"participants" bson:"participants"`
}

// Participant represents a user in a room.
type Participant struct {
	ID       string `json:"id" bson:"id"`
	Username string `json:"username" bson:"username"`
	Elo      int    `json:"elo" bson:"elo"`
}

// generateRoomID creates a random six-digit room ID as a string.
func generateRoomID() string {
	rand.Seed(time.Now().UnixNano())
	return strconv.Itoa(rand.Intn(900000) + 100000)
}

// CreateRoomHandler handles POST /rooms and creates a new debate room.
// CreateRoomHandler handles POST /rooms and creates a new debate room.
func CreateRoomHandler(c *gin.Context) {
	type CreateRoomInput struct {
		Type string `json:"type"` // public, private, invite
	}

	var input CreateRoomInput
	if err := c.ShouldBindJSON(&input); err != nil || input.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Get user email from middleware-set context
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: user email not found"})
		return
	}

	// Query user document using email
	userCollection := db.MongoClient.Database("DebateAI").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user struct {
		ID          primitive.ObjectID `bson:"_id"`
		Email       string `bson:"email"`
		DisplayName string `bson:"displayName"`
		Rating   int    `bson:"rating"`
	}

	err := userCollection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Add the room creator as the first participant
	creatorParticipant := Participant{
		ID:       user.ID.Hex(),
		Username: user.DisplayName,
		Elo:      user.Rating,
	}

	roomID := generateRoomID()
	newRoom := Room{
		ID:           roomID,
		Type:         input.Type,
		Participants: []Participant{creatorParticipant},
	}

	roomCollection := db.MongoClient.Database("DebateAI").Collection("rooms")
	_, err = roomCollection.InsertOne(ctx, newRoom)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create room"})
		return
	}

	c.JSON(http.StatusOK, newRoom)
}

// GetRoomsHandler handles GET /rooms and returns all rooms.
func GetRoomsHandler(c *gin.Context) {
	log.Println("🔍 GetRoomsHandler called")

	collection := db.MongoClient.Database("DebateAI").Collection("rooms")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching rooms"})
		return
	}

	var rooms []Room
	if err = cursor.All(ctx, &rooms); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding rooms"})
		return
	}

	c.JSON(http.StatusOK, rooms)
}

// JoinRoomHandler handles POST /rooms/:id/join where a user joins a room.
func JoinRoomHandler(c *gin.Context) {
	roomId := c.Param("id")
	
	// Get user email from middleware-set context
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: user email not found"})
		return
	}

	// Query user document using email
	userCollection := db.MongoClient.Database("DebateAI").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user struct {
		ID          primitive.ObjectID `bson:"_id"`
		Email       string `bson:"email"`
		DisplayName string `bson:"displayName"`
		Rating      int    `bson:"rating"`
	}

	err := userCollection.FindOne(ctx, bson.M{"email": email.(string)}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Create participant
	participant := Participant{
		ID:       user.ID.Hex(),
		Username: user.DisplayName,
		Elo:      user.Rating,
	}

	// Use atomic operation to join room
	roomCollection := db.MongoClient.Database("DebateAI").Collection("rooms")
	filter := bson.M{"_id": roomId}
	update := bson.M{
		"$addToSet": bson.M{"participants": participant},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updatedRoom Room
	if err := roomCollection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedRoom); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not join room"})
		return
	}

	// Remove user from matchmaking pool if they were in it
	matchmakingService := services.GetMatchmakingService()
	matchmakingService.RemoveFromPool(user.ID.Hex())

	c.JSON(http.StatusOK, updatedRoom)
}

// GetRoomParticipantsHandler handles GET /rooms/:id/participants and returns the participants of a room.
func GetRoomParticipantsHandler(c *gin.Context) {
	roomId := c.Param("id")
	
	// Get user email from middleware-set context
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: user email not found"})
		return
	}

	// Query room document
	roomCollection := db.MongoClient.Database("DebateAI").Collection("rooms")
	userCollection := db.MongoClient.Database("DebateAI").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var room Room
	err := roomCollection.FindOne(ctx, bson.M{"_id": roomId}).Decode(&room)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	// Get user ID from email
	var user struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	err = userCollection.FindOne(ctx, bson.M{"email": email.(string)}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if user is a participant in this room
	isParticipant := false
	for _, participant := range room.Participants {
		if participant.ID == user.ID.Hex() {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a participant in this room"})
		return
	}

	// Get full user details for each participant
	var participantsWithDetails []gin.H

	for _, participant := range room.Participants {
		var user struct {
			ID          primitive.ObjectID `bson:"_id"`
			Email       string `bson:"email"`
			DisplayName string `bson:"displayName"`
			Rating      int    `bson:"rating"`
			AvatarURL   string `bson:"avatarUrl"`
		}

		// Try to find user by ID first
		objectID, err := primitive.ObjectIDFromHex(participant.ID)
		if err != nil {
			// If not a valid ObjectID, try to find by email (fallback)
			err = userCollection.FindOne(ctx, bson.M{"email": participant.ID}).Decode(&user)
		} else {
			err = userCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
			if err != nil {
				// If not found by ID, try to find by email (fallback)
				err = userCollection.FindOne(ctx, bson.M{"email": participant.ID}).Decode(&user)
			}
		}
		
		if err != nil {
			// If user not found, use basic participant info with default avatar
			participantsWithDetails = append(participantsWithDetails, gin.H{
				"id":          participant.ID,
				"username":    participant.Username,
				"displayName": participant.Username,
				"elo":         participant.Elo,
				"avatarUrl":   "https://api.dicebear.com/9.x/adventurer/svg?seed=" + participant.Username,
			})
		} else {
			// Ensure we have a default avatar if none is set
			avatarUrl := user.AvatarURL
			if avatarUrl == "" {
				avatarUrl = "https://api.dicebear.com/9.x/adventurer/svg?seed=" + user.DisplayName
			}
			
			participantsWithDetails = append(participantsWithDetails, gin.H{
				"id":          user.ID.Hex(),
				"username":    user.DisplayName,
				"displayName": user.DisplayName,
				"elo":         user.Rating,
				"avatarUrl":   avatarUrl,
			})
		}
	}

	c.JSON(http.StatusOK, participantsWithDetails)
}
