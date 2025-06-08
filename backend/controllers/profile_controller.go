package controllers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"arguehub/db"
	"arguehub/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"fmt"
)

// calculateEloRating computes new Elo ratings after a match
func calculateEloRating(ratingA, ratingB int, scoreA float64) (newRatingA, newRatingB int) {
	const K = 32
	expectedA := 1.0 / (1.0 + pow(10, float64(ratingB-ratingA)/400.0))
	scoreB := 1.0 - scoreA
	expectedB := 1.0 - expectedA

	newRatingA = ratingA + int(float64(K)*(scoreA-expectedA))
	newRatingB = ratingB + int(float64(K)*(scoreB-expectedB))
	return newRatingA, newRatingB
}

// pow computes base^exponent as a simple helper
func pow(base, exponent float64) float64 {
	result := 1.0
	for i := 0; i < int(exponent); i++ {
		result *= base
	}
	return result
}

// GetProfile retrieves and returns user profile data, leaderboard, debate history, and stats
func GetProfile(c *gin.Context) {
	email := c.GetString("email")
	if email == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized", "message": "Missing email in context"})
		return
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Fetch user profile
	var user models.User
	err := db.MongoDatabase.Collection("users").FindOne(dbCtx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found", "message": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": fmt.Sprintf("Failed to fetch user: %v", err)})
		}
		return
	}

	// Set avatar URL with DiceBear fallback
	profileAvatarURL := user.AvatarURL
	profileName := user.DisplayName
	if profileAvatarURL == "" {
		if profileName == "" {
			profileName = extractNameFromEmail(email)
		}
		profileAvatarURL = "https://api.dicebear.com/9.x/adventurer/svg?seed=" + profileName
	}

	// Use "Steve" as default displayName if empty (optional, comment out if not desired)
	displayName := user.DisplayName
	if displayName == "" {
		displayName = "Steve"
	}

	// Fetch leaderboard
	leaderboardCursor, err := db.MongoDatabase.Collection("users").Find(
		dbCtx,
		bson.M{},
		options.Find().SetSort(bson.D{{"eloRating", -1}}).SetLimit(5), // Reduced to 5 to match frontend
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": fmt.Sprintf("Failed to fetch leaderboard: %v", err)})
		return
	}
	defer leaderboardCursor.Close(dbCtx)

	var leaderboard []struct {
		Rank        int    `json:"rank"`
		Name        string `json:"name"`
		Score       int    `json:"score"`
		AvatarUrl   string `json:"avatarUrl"`
		CurrentUser bool   `json:"currentUser"`
	}
	rank := 1
	for leaderboardCursor.Next(dbCtx) {
		var lbUser models.User
		if err := leaderboardCursor.Decode(&lbUser); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": fmt.Sprintf("Failed to decode leaderboard user: %v", err)})
			return
		}
		lbAvatarURL := lbUser.AvatarURL
		lbName := lbUser.DisplayName
		if lbAvatarURL == "" {
			if lbName == "" {
				lbName = extractNameFromEmail(lbUser.Email)
			}
			lbAvatarURL = "https://api.dicebear.com/9.x/adventurer/svg?seed=" + lbName
		}
		// Use "Steve" as default for leaderboard name if empty
		if lbName == "" {
			lbName = "Steve"
		}
		leaderboard = append(leaderboard, struct {
			Rank        int    `json:"rank"`
			Name        string `json:"name"`
			Score       int    `json:"score"`
			AvatarUrl   string `json:"avatarUrl"`
			CurrentUser bool   `json:"currentUser"`
		}{rank, lbName, lbUser.EloRating, lbAvatarURL, lbUser.Email == email})
		rank++
	}

	// Fetch debate history
	debateCursor, err := db.MongoDatabase.Collection("debates").Find(
		dbCtx,
		bson.M{"email": email},
		options.Find().SetSort(bson.D{{"date", -1}}).SetLimit(5),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": fmt.Sprintf("Failed to fetch debate history: %v", err)})
		return
	}
	defer debateCursor.Close(dbCtx)

	var debates []struct {
		Topic     string `bson:"topic" json:"topic"`
		Result    string `bson:"result" json:"result"`
		EloChange int    `bson:"eloChange" json:"eloChange"`
	}
	if err := debateCursor.All(dbCtx, &debates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": fmt.Sprintf("Failed to decode debate history: %v", err)})
		return
	}

	// Aggregate stats (wins, losses, draws)
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"email": email}}},
		{{"$group", bson.M{
			"_id":    nil,
			"wins":   bson.M{"$sum": bson.M{"$cond": bson.M{"if": bson.M{"$eq": []interface{}{"$result", "win"}}, "then": 1, "else": 0}}},
			"losses": bson.M{"$sum": bson.M{"$cond": bson.M{"if": bson.M{"$eq": []interface{}{"$result", "loss"}}, "then": 1, "else": 0}}},
			"draws":  bson.M{"$sum": bson.M{"$cond": bson.M{"if": bson.M{"$eq": []interface{}{"$result", "draw"}}, "then": 1, "else": 0}}},
		}}},
	}
	statsCursor, err := db.MongoDatabase.Collection("debates").Aggregate(dbCtx, pipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": fmt.Sprintf("Failed to aggregate stats: %v", err)})
		return
	}
	defer statsCursor.Close(dbCtx)

	var stats struct {
		Wins   int `json:"wins"`
		Losses int `json:"losses"`
		Draws  int `json:"draws"`
	}
	if statsCursor.Next(dbCtx) {
		if err := statsCursor.Decode(&stats); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": fmt.Sprintf("Failed to decode stats: %v", err)})
			return
		}
	}

	// Aggregate Elo history by month
	eloPipeline := mongo.Pipeline{
		{{"$match", bson.M{"email": email}}},
		{{"$sort", bson.M{"date": 1}}},
		{{"$group", bson.M{
			"_id":           bson.M{"$dateToString": bson.M{"format": "%Y-%m", "date": "$date"}},
			"lastEloChange": bson.M{"$last": "$eloChange"},
			"lastDate":      bson.M{"$last": "$date"},
		}}},
		{{"$sort", bson.M{"lastDate": 1}}},
	}
	eloCursor, err := db.MongoDatabase.Collection("debates").Aggregate(dbCtx, eloPipeline)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": fmt.Sprintf("Failed to aggregate Elo history: %v", err)})
		return
	}
	defer eloCursor.Close(dbCtx)

	var eloHistory []struct {
		Month string `json:"month"`
		Elo   int    `json:"elo"`
	}
	currentElo := user.EloRating
	for eloCursor.Next(dbCtx) {
		var result struct {
			ID            string    `bson:"_id"`
			LastEloChange int       `bson:"lastEloChange"`
			LastDate      time.Time `bson:"lastDate"`
		}
		if err := eloCursor.Decode(&result); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "message": fmt.Sprintf("Failed to decode Elo history: %v", err)})
			return
		}
		currentElo -= result.LastEloChange
		monthName := result.LastDate.Format("January")
		eloHistory = append(eloHistory, struct {
			Month string `json:"month"`
			Elo   int    `json:"elo"`
		}{monthName, currentElo})
	}
	// Add current Elo
	eloHistory = append(eloHistory, struct {
		Month string `json:"month"`
		Elo   int    `json:"elo"`
	}{time.Now().Format("January"), user.EloRating})

	// Construct response
	response := gin.H{
		"profile": gin.H{
			"displayName": displayName,
			"email":       user.Email,
			"bio":         user.Bio,
			"eloRating":   user.EloRating,
			"twitter":     user.Twitter,
			"instagram":   user.Instagram,
			"linkedin":    user.LinkedIn,
			"avatarUrl":   profileAvatarURL,
		},
		"leaderboard":   leaderboard,
		"debateHistory": debates,
		"stats": gin.H{
			"wins":       stats.Wins,
			"losses":     stats.Losses,
			"draws":      stats.Draws,
			"eloHistory": eloHistory,
		},
	}
	c.JSON(http.StatusOK, response)
}

func UpdateProfile(ctx *gin.Context) {
	email := ctx.GetString("email")
	if email == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized", "message": "Missing user email in context"})
		return
	}

	var updateData struct {
		DisplayName string `json:"displayName"`
		Bio         string `json:"bio"`
		Twitter     string `json:"twitter"`
		Instagram   string `json:"instagram"`
		LinkedIn    string `json:"linkedin"`
	}
	if err := ctx.ShouldBindJSON(&updateData); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "message": err.Error()})
		return
	}

	// Trim whitespace from input fields
	updateData.DisplayName = strings.TrimSpace(updateData.DisplayName)
	updateData.Bio = strings.TrimSpace(updateData.Bio)
	updateData.Twitter = strings.TrimSpace(updateData.Twitter)
	updateData.Instagram = strings.TrimSpace(updateData.Instagram)
	updateData.LinkedIn = strings.TrimSpace(updateData.LinkedIn)

	dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"email": email}
	update := bson.M{
		"$set": bson.M{
			"displayName": updateData.DisplayName,
			"bio":         updateData.Bio,
			"twitter":     updateData.Twitter,
			"instagram":   updateData.Instagram,
			"linkedin":    updateData.LinkedIn,
			"updatedAt":   time.Now(),
		},
	}
	result, err := db.MongoDatabase.Collection("users").UpdateOne(dbCtx, filter, update)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error", "message": "Failed to update profile"})
		return
	}

	if result.MatchedCount == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Not found", "message": "User not found"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

// UpdateEloAfterDebate updates Elo ratings for winner and loser
func UpdateEloAfterDebate(ctx *gin.Context) {
	var req struct {
		WinnerID string `json:"winnerId"`
		LoserID  string `json:"loserId"`
		Topic    string `json:"topic"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "message": err.Error()})
		return
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	winnerObjID, err := primitive.ObjectIDFromHex(req.WinnerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid winnerId"})
		return
	}
	loserObjID, err := primitive.ObjectIDFromHex(req.LoserID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid loserId"})
		return
	}

	var winner, loser models.User
	if err = db.MongoDatabase.Collection("users").FindOne(dbCtx, bson.M{"_id": winnerObjID}).Decode(&winner); err != nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Winner not found"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching winner from DB"})
		}
		return
	}

	if err = db.MongoDatabase.Collection("users").FindOne(dbCtx, bson.M{"_id": loserObjID}).Decode(&loser); err != nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Loser not found"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching loser from DB"})
		}
		return
	}

	// Calculate new Elo ratings
	newWinnerElo, newLoserElo := calculateEloRating(winner.EloRating, loser.EloRating, 1.0)
	winnerEloChange := newWinnerElo - winner.EloRating
	loserEloChange := newLoserElo - loser.EloRating

	// Update user Elo ratings
	_, err = db.MongoDatabase.Collection("users").UpdateOne(dbCtx, bson.M{"_id": winnerObjID}, bson.M{"$set": bson.M{"eloRating": newWinnerElo}})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	_, err = db.MongoDatabase.Collection("users").UpdateOne(dbCtx, bson.M{"_id": loserObjID}, bson.M{"$set": bson.M{"eloRating": newLoserElo}})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Record debate results
	now := time.Now()
	winnerDebate := models.Debate{
		Email:     winner.Email,
		Topic:     req.Topic,
		Result:    "win",
		EloChange: winnerEloChange,
		Date:      now,
	}
	loserDebate := models.Debate{
		Email:     loser.Email,
		Topic:     req.Topic,
		Result:    "loss",
		EloChange: loserEloChange,
		Date:      now,
	}

	db.MongoDatabase.Collection("debates").InsertOne(dbCtx, winnerDebate)
	db.MongoDatabase.Collection("debates").InsertOne(dbCtx, loserDebate)

	ctx.JSON(http.StatusOK, gin.H{
		"winnerNewElo": newWinnerElo,
		"loserNewElo":  newLoserElo,
	})
}

// extractNameFromEmail extracts the name from an email address
func extractNameFromEmail(email string) string {
	for i, char := range email {
		if char == '@' {
			return email[:i]
		}
	}
	return email
}
