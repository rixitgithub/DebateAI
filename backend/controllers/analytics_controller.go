package controllers

import (
	"arguehub/db"
	"arguehub/models"
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetAnalytics returns current analytics snapshot
func GetAnalytics(ctx *gin.Context) {
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Get total debates
	debatesCollection := db.MongoDatabase.Collection("debates")
	totalDebates, _ := debatesCollection.CountDocuments(dbCtx, bson.M{})

	// Get bot debates
	botDebatesCollection := db.MongoDatabase.Collection("debates_vs_bot")
	totalBotDebates, _ := botDebatesCollection.CountDocuments(dbCtx, bson.M{})
	totalDebates += totalBotDebates

	// Get debates today
	debatesToday, _ := debatesCollection.CountDocuments(dbCtx, bson.M{
		"date": bson.M{"$gte": todayStart},
	})

	// Get active users (users active in last 30 days)
	usersCollection := db.MongoDatabase.Collection("users")
	activeUsers, _ := usersCollection.CountDocuments(dbCtx, bson.M{
		"updatedAt": bson.M{"$gte": thirtyDaysAgo},
	})

	// Get total users
	totalUsers, _ := usersCollection.CountDocuments(dbCtx, bson.M{})

	// Get new users today
	newUsersToday, _ := usersCollection.CountDocuments(dbCtx, bson.M{
		"createdAt": bson.M{"$gte": todayStart},
	})

	// Get total comments (team debate messages + team chat messages)
	teamDebateMessagesCollection := db.MongoDatabase.Collection("team_debate_messages")
	totalTeamDebateMessages, _ := teamDebateMessagesCollection.CountDocuments(dbCtx, bson.M{})

	teamChatMessagesCollection := db.MongoDatabase.Collection("team_chat_messages")
	totalTeamChatMessages, _ := teamChatMessagesCollection.CountDocuments(dbCtx, bson.M{})

	totalComments := totalTeamDebateMessages + totalTeamChatMessages

	// Get comments today
	commentsToday, _ := teamDebateMessagesCollection.CountDocuments(dbCtx, bson.M{
		"timestamp": bson.M{"$gte": todayStart},
	})
	chatCommentsToday, _ := teamChatMessagesCollection.CountDocuments(dbCtx, bson.M{
		"timestamp": bson.M{"$gte": todayStart},
	})
	commentsToday += chatCommentsToday

	// Create analytics snapshot
	snapshot := models.AnalyticsSnapshot{
		ID:            primitive.NewObjectID(),
		Timestamp:     now,
		TotalDebates:  totalDebates,
		ActiveUsers:   activeUsers,
		TotalComments: totalComments,
		TotalUsers:    totalUsers,
		DebatesToday:  debatesToday,
		CommentsToday: commentsToday,
		NewUsersToday: newUsersToday,
	}

	// Save snapshot to database (optional, for historical tracking)
	snapshotsCollection := db.MongoDatabase.Collection("analytics_snapshots")
	snapshotsCollection.InsertOne(dbCtx, snapshot)

	ctx.JSON(http.StatusOK, gin.H{
		"totalDebates":  snapshot.TotalDebates,
		"activeUsers":   snapshot.ActiveUsers,
		"totalComments": snapshot.TotalComments,
		"totalUsers":    snapshot.TotalUsers,
		"debatesToday":  snapshot.DebatesToday,
		"commentsToday": snapshot.CommentsToday,
		"newUsersToday": snapshot.NewUsersToday,
		"timestamp":     snapshot.Timestamp.Format(time.RFC3339),
	})
}

// GetAnalyticsHistory returns analytics data over time
func GetAnalyticsHistory(ctx *gin.Context) {
	days := 7 // default to 7 days
	if daysStr := ctx.Query("days"); daysStr != "" {
		if parsedDays, err := strconv.Atoi(daysStr); err == nil && parsedDays > 0 {
			days = parsedDays
		}
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now()
	startDate := now.AddDate(0, 0, -days)

	snapshotsCollection := db.MongoDatabase.Collection("analytics_snapshots")
	
	opts := options.Find().SetSort(bson.M{"timestamp": 1})
	cursor, err := snapshotsCollection.Find(dbCtx, bson.M{
		"timestamp": bson.M{"$gte": startDate},
	}, opts)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch analytics history", "message": err.Error()})
		return
	}
	defer cursor.Close(dbCtx)

	var existingSnapshots []models.AnalyticsSnapshot
	if err := cursor.All(dbCtx, &existingSnapshots); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode snapshots", "message": err.Error()})
		return
	}

	// Create a map of existing snapshots by day
	snapshotMap := make(map[string]models.AnalyticsSnapshot)
	for _, snapshot := range existingSnapshots {
		dayKey := snapshot.Timestamp.Format("2006-01-02")
		snapshotMap[dayKey] = snapshot
	}

	// Generate snapshots for all days in the requested period
	// Use existing snapshots where available, otherwise generate from actual data
	var snapshots []models.AnalyticsSnapshot
	for i := 0; i < days; i++ {
		date := startDate.AddDate(0, 0, i)
		dateStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		dateEnd := dateStart.AddDate(0, 0, 1)
		dayKey := dateStart.Format("2006-01-02")

		var snapshot models.AnalyticsSnapshot
		if existingSnapshot, exists := snapshotMap[dayKey]; exists {
			// Use existing snapshot
			snapshot = existingSnapshot
		} else {
			// Generate snapshot from actual data for this day
			// Count debates for this day (including bot debates)
			debatesCollection := db.MongoDatabase.Collection("debates")
			debatesCount, err := debatesCollection.CountDocuments(dbCtx, bson.M{
				"date": bson.M{"$gte": dateStart, "$lt": dateEnd},
			})
			if err != nil {
				log.Printf("Error counting debates for %s: %v", dayKey, err)
			}
			
			botDebatesCollection := db.MongoDatabase.Collection("debates_vs_bot")
			// Bot debates use createdAt (int64 Unix timestamp) instead of date
			dateStartUnix := dateStart.Unix()
			dateEndUnix := dateEnd.Unix()
			botDebatesCount, err := botDebatesCollection.CountDocuments(dbCtx, bson.M{
				"createdAt": bson.M{"$gte": dateStartUnix, "$lt": dateEndUnix},
			})
			if err != nil {
				log.Printf("Error counting bot debates for %s: %v", dayKey, err)
			}
			debatesCount += botDebatesCount
			
			log.Printf("Generated snapshot for %s: %d debates (%d regular + %d bot)", dayKey, debatesCount, debatesCount-botDebatesCount, botDebatesCount)

			// Count comments for this day
			teamDebateMessagesCollection := db.MongoDatabase.Collection("team_debate_messages")
			commentsCount, _ := teamDebateMessagesCollection.CountDocuments(dbCtx, bson.M{
				"timestamp": bson.M{"$gte": dateStart, "$lt": dateEnd},
			})
			
			teamChatMessagesCollection := db.MongoDatabase.Collection("team_chat_messages")
			chatCommentsCount, _ := teamChatMessagesCollection.CountDocuments(dbCtx, bson.M{
				"timestamp": bson.M{"$gte": dateStart, "$lt": dateEnd},
			})
			commentsCount += chatCommentsCount

			// Count new users for this day
			usersCollection := db.MongoDatabase.Collection("users")
			newUsersCount, _ := usersCollection.CountDocuments(dbCtx, bson.M{
				"createdAt": bson.M{"$gte": dateStart, "$lt": dateEnd},
			})

			snapshot = models.AnalyticsSnapshot{
				ID:            primitive.NewObjectID(),
				Timestamp:     dateStart,
				DebatesToday:  debatesCount,
				CommentsToday: commentsCount,
				NewUsersToday: newUsersCount,
			}
		}
		snapshots = append(snapshots, snapshot)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"snapshots": snapshots,
		"days":      days,
	})
}

// GetAdminActionLogs returns admin action logs
func GetAdminActionLogs(ctx *gin.Context) {
	page := 1
	limit := 50

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	skip := (page - 1) * limit

	logsCollection := db.MongoDatabase.Collection("admin_action_logs")
	opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit)).SetSort(bson.M{"timestamp": -1})
	cursor, err := logsCollection.Find(dbCtx, bson.M{}, opts)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch logs", "message": err.Error()})
		return
	}
	defer cursor.Close(dbCtx)

	var logs []models.AdminActionLog
	if err := cursor.All(dbCtx, &logs); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode logs", "message": err.Error()})
		return
	}

	total, _ := logsCollection.CountDocuments(dbCtx, bson.M{})

	ctx.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

