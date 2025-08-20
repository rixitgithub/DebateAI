package services

import (
	"testing"
	"time"
)

func TestMatchmakingService(t *testing.T) {
	// Get the singleton service
	ms := GetMatchmakingService()

	// Test adding users to pool
	err := ms.AddToPool("user1", "Alice", 1200)
	if err != nil {
		t.Errorf("Failed to add user1 to pool: %v", err)
	}

	err = ms.AddToPool("user2", "Bob", 1250)
	if err != nil {
		t.Errorf("Failed to add user2 to pool: %v", err)
	}

	// Test getting pool
	pool := ms.GetPool()
	if len(pool) != 2 {
		t.Errorf("Expected 2 users in pool, got %d", len(pool))
	}

	// Test removing user from pool
	ms.RemoveFromPool("user1")
	pool = ms.GetPool()
	if len(pool) != 1 {
		t.Errorf("Expected 1 user in pool after removal, got %d", len(pool))
	}

	// Test updating activity
	ms.UpdateActivity("user2")
	pool = ms.GetPool()
	if len(pool) != 1 {
		t.Errorf("Expected 1 user in pool after activity update, got %d", len(pool))
	}
}

func TestEloTolerance(t *testing.T) {
	ms := GetMatchmakingService()

	// Add users with different Elo ratings
	ms.AddToPool("user1", "Alice", 1200)  // Range: 1000-1400
	ms.AddToPool("user2", "Bob", 1250)    // Range: 1050-1450
	ms.AddToPool("user3", "Charlie", 800) // Range: 600-1000 (should not match)

	// Wait a bit for matching to happen
	time.Sleep(100 * time.Millisecond)

	// Check pool - user1 and user2 should be matched and removed
	pool := ms.GetPool()
	if len(pool) != 1 {
		t.Errorf("Expected 1 user remaining in pool (Charlie), got %d", len(pool))
	}

	if pool[0].Username != "Charlie" {
		t.Errorf("Expected Charlie to remain in pool, got %s", pool[0].Username)
	}
}
