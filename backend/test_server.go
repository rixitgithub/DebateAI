package main

import (
	"log"
	"time"
	
	"arguehub/services"
)

func main() {
	log.Println("Testing matchmaking service...")
	
	// Test the matchmaking service
	ms := services.GetMatchmakingService()
	
	// Add some test users (but don't start matchmaking yet)
	err := ms.AddToPool("user1", "Alice", 1200)
	if err != nil {
		log.Printf("Error adding user1: %v", err)
	}
	
	err = ms.AddToPool("user2", "Bob", 1250)
	if err != nil {
		log.Printf("Error adding user2: %v", err)
	}
	
	// Check pool - should be empty since no one started matchmaking
	pool := ms.GetPool()
	log.Printf("Pool size after adding users (no matchmaking): %d", len(pool))
	
	// Start matchmaking for both users
	err = ms.StartMatchmaking("user1")
	if err != nil {
		log.Printf("Error starting matchmaking for user1: %v", err)
	}
	
	err = ms.StartMatchmaking("user2")
	if err != nil {
		log.Printf("Error starting matchmaking for user2: %v", err)
	}
	
	// Check pool - should have 2 users now
	pool = ms.GetPool()
	log.Printf("Pool size after starting matchmaking: %d", len(pool))
	
	for _, user := range pool {
		log.Printf("User in pool: %s (Elo: %d)", user.Username, user.Elo)
	}
	
	// Wait a bit for matching
	time.Sleep(1 * time.Second)
	
	// Check pool after matching
	pool = ms.GetPool()
	log.Printf("Pool size after matching: %d", len(pool))
	
	for _, user := range pool {
		log.Printf("User remaining in pool: %s (Elo: %d)", user.Username, user.Elo)
	}
	
	log.Println("Test completed successfully!")
}
