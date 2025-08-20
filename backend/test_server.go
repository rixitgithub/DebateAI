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
	
	// Add some test users
	err := ms.AddToPool("user1", "Alice", 1200)
	if err != nil {
		log.Printf("Error adding user1: %v", err)
	}
	
	err = ms.AddToPool("user2", "Bob", 1250)
	if err != nil {
		log.Printf("Error adding user2: %v", err)
	}
	
	// Wait a bit for matching
	time.Sleep(1 * time.Second)
	
	// Check pool
	pool := ms.GetPool()
	log.Printf("Pool size: %d", len(pool))
	
	for _, user := range pool {
		log.Printf("User in pool: %s (Elo: %d)", user.Username, user.Elo)
	}
	
	log.Println("Test completed successfully!")
}
