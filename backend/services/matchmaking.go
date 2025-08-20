package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"arguehub/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MatchmakingPool represents a user in the matchmaking queue
type MatchmakingPool struct {
	UserID       string    `json:"userId" bson:"userId"`
	Username     string    `json:"username" bson:"username"`
	Elo          int       `json:"elo" bson:"elo"`
	MinElo       int       `json:"minElo" bson:"minElo"`
	MaxElo       int       `json:"maxElo" bson:"maxElo"`
	JoinedAt     time.Time `json:"joinedAt" bson:"joinedAt"`
	LastActivity time.Time `json:"lastActivity" bson:"lastActivity"`
}

// MatchmakingService handles the matchmaking logic
type MatchmakingService struct {
	pool map[string]*MatchmakingPool
	mutex sync.RWMutex
}

var (
	matchmakingService *MatchmakingService
	once sync.Once
)

// GetMatchmakingService returns the singleton matchmaking service
func GetMatchmakingService() *MatchmakingService {
	once.Do(func() {
		matchmakingService = &MatchmakingService{
			pool: make(map[string]*MatchmakingPool),
		}
		go matchmakingService.cleanupInactiveUsers()
	})
	return matchmakingService
}

// AddToPool adds a user to the matchmaking pool
func (ms *MatchmakingService) AddToPool(userID, username string, elo int) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	// Calculate Elo tolerance (default ±200, but can be adjusted based on user preferences)
	eloTolerance := 200
	minElo := elo - eloTolerance
	maxElo := elo + eloTolerance

	poolEntry := &MatchmakingPool{
		UserID:       userID,
		Username:     username,
		Elo:          elo,
		MinElo:       minElo,
		MaxElo:       maxElo,
		JoinedAt:     time.Now(),
		LastActivity: time.Now(),
	}

	ms.pool[userID] = poolEntry
	log.Printf("User %s (Elo: %d) added to matchmaking pool", username, elo)

	// Try to find a match immediately
	go ms.findMatch(userID)
	return nil
}

// RemoveFromPool removes a user from the matchmaking pool
func (ms *MatchmakingService) RemoveFromPool(userID string) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	if _, exists := ms.pool[userID]; exists {
		delete(ms.pool, userID)
		log.Printf("User %s removed from matchmaking pool", userID)
	}
}

// UpdateActivity updates the last activity time for a user
func (ms *MatchmakingService) UpdateActivity(userID string) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	
	if poolEntry, exists := ms.pool[userID]; exists {
		poolEntry.LastActivity = time.Now()
	}
}

// GetPool returns a copy of the current matchmaking pool
func (ms *MatchmakingService) GetPool() []*MatchmakingPool {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()
	
	pool := make([]*MatchmakingPool, 0, len(ms.pool))
	for _, entry := range ms.pool {
		pool = append(pool, entry)
	}
	return pool
}

// findMatch attempts to find a suitable opponent for the given user
func (ms *MatchmakingService) findMatch(userID string) {
	ms.mutex.RLock()
	user, exists := ms.pool[userID]
	if !exists {
		ms.mutex.RUnlock()
		return
	}
	ms.mutex.RUnlock()

	// Find potential opponents
	var bestMatch *MatchmakingPool
	var bestScore float64 = -1

	ms.mutex.RLock()
	for _, opponent := range ms.pool {
		if opponent.UserID == userID {
			continue // Skip self
		}

		// Check if Elo ranges overlap
		if user.MinElo <= opponent.MaxElo && user.MaxElo >= opponent.MinElo {
			// Calculate match quality score (lower is better)
			eloDiff := math.Abs(float64(user.Elo - opponent.Elo))
			waitTime := time.Since(opponent.JoinedAt).Seconds()
			
			// Score based on Elo difference and wait time
			score := eloDiff - (waitTime * 0.1) // Prefer closer Elo, but consider wait time
			
			if bestMatch == nil || score < bestScore {
				bestMatch = opponent
				bestScore = score
			}
		}
	}
	ms.mutex.RUnlock()

	if bestMatch != nil {
		// Create a room for these two users
		ms.createRoomForMatch(user, bestMatch)
	}
}

// createRoomForMatch creates a room for two matched users
func (ms *MatchmakingService) createRoomForMatch(user1, user2 *MatchmakingPool) {
	// Use atomic operation to create room and remove users from pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	roomCollection := db.MongoClient.Database("DebateAI").Collection("rooms")
	
	// Generate room ID
	roomID := generateRoomID()
	
	// Create room with both participants
	room := bson.M{
		"_id": roomID,
		"type": "public",
		"participants": []bson.M{
			{
				"id":       user1.UserID,
				"username": user1.Username,
				"elo":      user1.Elo,
			},
			{
				"id":       user2.UserID,
				"username": user2.Username,
				"elo":      user2.Elo,
			},
		},
		"createdAt": time.Now(),
		"status":    "waiting", // waiting, active, completed
	}

	// Use findOneAndUpdate to atomically create the room
	opts := options.FindOneAndUpdate().SetUpsert(true)
	var result bson.M
	err := roomCollection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": roomID},
		bson.M{"$setOnInsert": room},
		opts,
	).Decode(&result)

	if err != nil {
		log.Printf("Failed to create room for match: %v", err)
		return
	}

	// Remove both users from the pool
	ms.RemoveFromPool(user1.UserID)
	ms.RemoveFromPool(user2.UserID)

	log.Printf("Created room %s for users %s (Elo: %d) and %s (Elo: %d)", 
		roomID, user1.Username, user1.Elo, user2.Username, user2.Elo)

	// Broadcast room creation to WebSocket clients
	// This will be handled by the WebSocket handler when it detects the new room
	log.Printf("Room %s created successfully, broadcasting to WebSocket clients", roomID)
}

// cleanupInactiveUsers removes users who have been inactive for too long
func (ms *MatchmakingService) cleanupInactiveUsers() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ms.mutex.Lock()
		now := time.Now()
		for userID, poolEntry := range ms.pool {
			// Remove users inactive for more than 5 minutes
			if now.Sub(poolEntry.LastActivity) > 5*time.Minute {
				delete(ms.pool, userID)
				log.Printf("Removed inactive user %s from matchmaking pool", userID)
			}
		}
		ms.mutex.Unlock()
	}
}

// generateRoomID creates a random six-digit room ID
func generateRoomID() string {
	// Simple implementation - in production, ensure uniqueness
	return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
}
