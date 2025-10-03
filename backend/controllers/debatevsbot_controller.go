package controllers

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"arguehub/db"
	"arguehub/models"
	"arguehub/services"
	"arguehub/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DebateRequest struct {
	BotName      string           `json:"botName" binding:"required"`
	BotLevel     string           `json:"botLevel" binding:"required"`
	Topic        string           `json:"topic" binding:"required"`
	Stance       string           `json:"stance" binding:"required"`
	History      []models.Message `json:"history"`
	PhaseTimings []PhaseTiming    `json:"phaseTimings"`
	Context      string           `json:"context"`
}

type PhaseTiming struct {
	Name string `json:"name" binding:"required"`
	Time int    `json:"time" binding:"required"` // Single time value in seconds
}

type JudgeRequest struct {
	History []models.Message `json:"history" binding:"required"`
}

type DebateResponse struct {
	DebateId     string               `json:"debateId"`
	BotName      string               `json:"botName"`
	BotLevel     string               `json:"botLevel"`
	Topic        string               `json:"topic"`
	Stance       string               `json:"stance"`
	PhaseTimings []models.PhaseTiming `json:"phaseTimings,omitempty"` // Backend format
}

type DebateMessageResponse struct {
	DebateId string `json:"debateId"`
	BotName  string `json:"botName"`
	BotLevel string `json:"botLevel"`
	Topic    string `json:"topic"`
	Stance   string `json:"stance"`
	Response string `json:"response"`
}

type JudgeResponse struct {
	Result string `json:"result"`
}

func CreateDebate(c *gin.Context) {
	// Extract token from request header
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(401, gin.H{"error": "Authorization token required"})
		return
	}

	token = strings.TrimPrefix(token, "Bearer ")
	// Validate token and get user email
	valid, email, err := utils.ValidateTokenAndFetchEmail("./config/config.prod.yml", token, c)
	if err != nil || !valid {
		c.JSON(401, gin.H{"error": "Invalid or expired token"})
		return
	}

	var req DebateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Convert PhaseTimings to backend model format
	backendPhaseTimings := make([]models.PhaseTiming, len(req.PhaseTimings))
	for i, pt := range req.PhaseTimings {
		backendPhaseTimings[i] = models.PhaseTiming{
			Name:     pt.Name,
			UserTime: pt.Time,
			BotTime:  pt.Time,
		}
	}

	debate := models.DebateVsBot{
		Email:        email,
		BotName:      req.BotName,
		BotLevel:     req.BotLevel,
		Topic:        req.Topic,
		Stance:       req.Stance,
		History:      req.History,
		PhaseTimings: backendPhaseTimings,
		CreatedAt:    time.Now().Unix(),
	}

	debateID, err := services.CreateDebateService(&debate, req.Stance)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create debate: " + err.Error()})
		return
	}

	response := DebateResponse{
		DebateId:     debateID,
		BotName:      req.BotName,
		BotLevel:     req.BotLevel,
		Topic:        req.Topic,
		Stance:       req.Stance,
		PhaseTimings: backendPhaseTimings,
	}
	c.JSON(200, response)
}

func SendDebateMessage(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(401, gin.H{"error": "Authorization token required"})
		return
	}

	token = strings.TrimPrefix(token, "Bearer ")
	valid, email, err := utils.ValidateTokenAndFetchEmail("./config/config.prod.yml", token, c)
	if err != nil || !valid {
		c.JSON(401, gin.H{"error": "Invalid or expired token"})
		return
	}

	var req DebateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Generate bot response with the additional context field.
	botResponse := services.GenerateBotResponse(req.BotName, req.BotLevel, req.Topic, req.History, req.Stance, req.Context, 150)

	// Update debate history with the bot's response.
	updatedHistory := append(req.History, models.Message{
		Sender: "Bot",
		Text:   botResponse,
		// You can also store the phase if needed.
	})

	debate := models.DebateVsBot{
		Email:     email,
		BotName:   req.BotName,
		BotLevel:  req.BotLevel,
		Topic:     req.Topic,
		Stance:    req.Stance,
		History:   updatedHistory,
		CreatedAt: time.Now().Unix(),
	}

	// Save to database (assuming ID is generated in service or here)
	if debate.ID.IsZero() {
		debate.ID = primitive.NewObjectID()
	}
	if err := db.SaveDebateVsBot(debate); err != nil {
		log.Printf("Failed to save debate: %v", err)
	}

	response := DebateMessageResponse{
		DebateId: debate.ID.Hex(),
		BotName:  req.BotName,
		BotLevel: req.BotLevel,
		Topic:    req.Topic,
		Stance:   req.Stance,
		Response: botResponse,
	}
	c.JSON(200, response)
}

func JudgeDebate(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(401, gin.H{"error": "Authorization token required"})
		return
	}

	token = strings.TrimPrefix(token, "Bearer ")
	valid, email, err := utils.ValidateTokenAndFetchEmail("./config/config.prod.yml", token, c)
	if err != nil || !valid {
		c.JSON(401, gin.H{"error": "Invalid or expired token"})
		return
	}

	// Get user ID from database using email
	userID, err := utils.GetUserIDFromEmail(email)
	if err != nil {
		c.JSON(401, gin.H{"error": "Failed to get user ID"})
		return
	}

	var req JudgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Judge the debate
	result := services.JudgeDebate(req.History)

	// Update debate outcome
	if err := db.UpdateDebateVsBotOutcome(email, result); err != nil {
		log.Printf("Failed to update debate outcome: %v", err)
	}

	// Get the latest debate information to extract proper details
	latestDebate, err := db.GetLatestDebateVsBot(email)
	if err != nil {
		log.Printf("Failed to get latest debate info: %v", err)
		// Use defaults if we can't get the debate info
		latestDebate = &models.DebateVsBot{
			Topic:   "Debate vs Bot",
			BotName: "AI Bot",
		}
	}

	// Determine result from judge's response
	log.Printf("Raw judge result for bot debate: %s", result)
	resultStatus := "pending"
	
	// Try to parse JSON response first
	var judgeResponse map[string]interface{}
	if err := json.Unmarshal([]byte(result), &judgeResponse); err == nil {
		// If JSON parsing succeeds, extract winner from verdict
		log.Printf("Successfully parsed JSON response for bot debate: %+v", judgeResponse)
		if verdict, ok := judgeResponse["verdict"].(map[string]interface{}); ok {
			if winner, ok := verdict["winner"].(string); ok {
				log.Printf("Extracted winner for bot debate: %s", winner)
				if strings.EqualFold(winner, "User") {
					resultStatus = "win"
					log.Printf("User wins bot debate")
				} else if strings.EqualFold(winner, "Bot") {
					resultStatus = "loss"
					log.Printf("Bot wins bot debate")
				} else if strings.EqualFold(winner, "Draw") {
					resultStatus = "draw"
					log.Printf("Bot debate is a draw")
				} else {
					// If winner is not clearly "User", "Bot", or "Draw", default to loss
					resultStatus = "loss"
					log.Printf("Winner unclear for bot debate, defaulting to loss")
				}
			} else {
				log.Printf("Winner field not found in verdict or not a string for bot debate")
				// Default to loss if we can't determine winner
				resultStatus = "loss"
			}
		} else {
			log.Printf("Verdict field not found in response or not a map for bot debate")
			// Default to loss if we can't determine winner
			resultStatus = "loss"
		}
	} else {
		// Fallback to string matching if JSON parsing fails
		log.Printf("JSON parsing failed for bot debate: %v, falling back to string matching", err)
		resultLower := strings.ToLower(result)
		if strings.Contains(resultLower, "user win") || strings.Contains(resultLower, "user wins") || 
		   strings.Contains(resultLower, "user") && strings.Contains(resultLower, "win") {
			resultStatus = "win"
			log.Printf("String matching: User wins bot debate")
		} else if strings.Contains(resultLower, "bot win") || strings.Contains(resultLower, "bot wins") || 
			strings.Contains(resultLower, "lose") || strings.Contains(resultLower, "loss") ||
			strings.Contains(resultLower, "bot") && strings.Contains(resultLower, "win") {
			resultStatus = "loss"
			log.Printf("String matching: Bot wins bot debate")
		} else if strings.Contains(resultLower, "draw") {
			resultStatus = "draw"
			log.Printf("String matching: Bot debate is a draw")
		} else {
			// If no clear pattern is found, default to loss
			resultStatus = "loss"
			log.Printf("String matching: No clear winner pattern found, defaulting to loss")
		}
	}
	
	log.Printf("Final result status for bot debate: %s", resultStatus)

	// Save transcript with proper debate information
	err = services.SaveDebateTranscript(
		userID,
		email,
		"user_vs_bot",
		latestDebate.Topic,
		latestDebate.BotName,
		resultStatus,
		req.History,
		nil,
	)
	if err != nil {
		log.Printf("Failed to save debate transcript: %v", err)
	} else {
		log.Printf("Successfully saved debate transcript for user %s: %s vs %s, Result: %s", 
			email, latestDebate.Topic, latestDebate.BotName, resultStatus)
	}

	c.JSON(200, JudgeResponse{
		Result: result,
	})
}


