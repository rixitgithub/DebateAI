package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"arguehub/db"
	"arguehub/models"

	"github.com/google/generative-ai-go/genai"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func SubmitTranscripts(roomID, role, email string, transcripts map[string]string) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Collections
	transcriptCollection := db.MongoDatabase.Collection("debate_transcripts")
	resultCollection := db.MongoDatabase.Collection("debate_results")

	// Check if a judgment result already exists for this room
	var existingResult models.DebateResult
	err := resultCollection.FindOne(ctx, bson.M{"roomId": roomID}).Decode(&existingResult)
	if err == nil {
		// Judgment already exists, return it
		return map[string]interface{}{
			"message": "Debate already judged",
			"result":  existingResult.Result,
		}, nil
	}
	if err != mongo.ErrNoDocuments {
		return nil, errors.New("failed to check existing result: " + err.Error())
	}

	// No judgment exists yet, proceed with transcript submission
	filter := bson.M{"roomId": roomID, "role": role}
	var existingTranscript models.DebateTranscript
	err = transcriptCollection.FindOne(ctx, filter).Decode(&existingTranscript)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, errors.New("failed to check existing submission: " + err.Error())
	}

	if err == nil {
		// Update existing submission
		update := bson.M{
			"$set": bson.M{
				"transcripts": transcripts,
				"updatedAt":   time.Now(),
			},
		}
		_, err = transcriptCollection.UpdateOne(ctx, filter, update)
		if err != nil {
			return nil, errors.New("failed to update submission: " + err.Error())
		}
	} else {
		// Insert new submission
		doc := models.DebateTranscript{
			RoomID:      roomID,
			Role:        role,
			Email:       email,
			Transcripts: transcripts,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		_, err = transcriptCollection.InsertOne(ctx, doc)
		if err != nil {
			return nil, errors.New("failed to insert submission: " + err.Error())
		}
	}

	// Check if both sides have submitted
	var forSubmission, againstSubmission models.DebateTranscript
	errFor := transcriptCollection.FindOne(ctx, bson.M{"roomId": roomID, "role": "for"}).Decode(&forSubmission)
	errAgainst := transcriptCollection.FindOne(ctx, bson.M{"roomId": roomID, "role": "against"}).Decode(&againstSubmission)

	if errFor == nil && errAgainst == nil {
		// Both submissions exist, compute judgment once
		merged := mergeTranscripts(forSubmission.Transcripts, againstSubmission.Transcripts)
		result := JudgeDebateHumanVsHuman(merged)

		// Store the result
		resultDoc := models.DebateResult{
			RoomID:    roomID,
			Result:    result,
			CreatedAt: time.Now(),
		}
		_, err = resultCollection.InsertOne(ctx, resultDoc)
		if err != nil {
			log.Printf("Failed to store debate result: %v", err)
			return nil, errors.New("failed to store debate result: " + err.Error())
		}

			// Save the debate transcript for both users
	// First, get user IDs for both participants
	userCollection := db.MongoDatabase.Collection("users")
	savedTranscriptsCollection := db.MongoDatabase.Collection("saved_debate_transcripts")
	
	var forUser, againstUser models.User
	errFor := userCollection.FindOne(ctx, bson.M{"email": forSubmission.Email}).Decode(&forUser)
	errAgainst := userCollection.FindOne(ctx, bson.M{"email": againstSubmission.Email}).Decode(&againstUser)
	
	if errFor == nil && errAgainst == nil {
		// Check if transcripts have already been saved for this room to prevent duplicates
		var existingTranscript models.SavedDebateTranscript
		err = savedTranscriptsCollection.FindOne(ctx, bson.M{
			"topic": "User vs User Debate",
			"$or": []bson.M{
				{"userId": forUser.ID, "opponent": againstUser.Email},
				{"userId": againstUser.ID, "opponent": forUser.Email},
			},
			"createdAt": bson.M{"$gte": time.Now().Add(-5 * time.Minute)}, // Check for recent transcripts (within 5 minutes)
		}).Decode(&existingTranscript)
		
		if err == nil {
			// Transcript already exists, skip saving to prevent duplicates
			log.Printf("Transcript already exists for room %s, skipping duplicate save", roomID)
		} else if err == mongo.ErrNoDocuments {
			// No existing transcript found, proceed with saving
			// Determine result for each user
			resultFor := "pending"
			resultAgainst := "pending"
			
			// Try to parse the JSON response to extract the winner
			log.Printf("Raw judge result: %s", result)
			var judgeResponse map[string]interface{}
			if err := json.Unmarshal([]byte(result), &judgeResponse); err == nil {
				// If JSON parsing succeeds, extract winner from verdict
				log.Printf("Successfully parsed JSON response: %+v", judgeResponse)
				if verdict, ok := judgeResponse["verdict"].(map[string]interface{}); ok {
					if winner, ok := verdict["winner"].(string); ok {
						log.Printf("Extracted winner: %s", winner)
						if strings.EqualFold(winner, "For") {
							resultFor = "win"
							resultAgainst = "loss"
							log.Printf("For side wins, Against side loses")
						} else if strings.EqualFold(winner, "Against") {
							resultFor = "loss"
							resultAgainst = "win"
							log.Printf("Against side wins, For side loses")
						} else {
							// If winner is not clearly "For" or "Against", treat as draw
							resultFor = "draw"
							resultAgainst = "draw"
							log.Printf("Winner unclear, treating as draw")
						}
					} else {
						log.Printf("Winner field not found in verdict or not a string")
					}
				} else {
					log.Printf("Verdict field not found in response or not a map")
				}
			} else {
				// Fallback to string matching if JSON parsing fails
				log.Printf("JSON parsing failed: %v, falling back to string matching", err)
				resultLower := strings.ToLower(result)
				if strings.Contains(resultLower, "for") {
					resultFor = "win"
					resultAgainst = "loss"
					log.Printf("String matching: For side wins")
				} else if strings.Contains(resultLower, "against") {
					resultFor = "loss"
					resultAgainst = "win"
					log.Printf("String matching: Against side wins")
				} else {
					resultFor = "draw"
					resultAgainst = "draw"
					log.Printf("String matching: No clear winner, treating as draw")
				}
			}
			
			log.Printf("Final results - For: %s, Against: %s", resultFor, resultAgainst)
			
			// Extract topic from transcripts (you might need to adjust this based on your data structure)
			topic := "User vs User Debate"
			
			// Save transcript for "for" user
			err = SaveDebateTranscript(
				forUser.ID,
				forUser.Email,
				"user_vs_user",
				topic,
				againstUser.Email,
				resultFor,
				[]models.Message{}, // You might want to reconstruct messages from transcripts
				forSubmission.Transcripts,
			)
			if err != nil {
				log.Printf("Failed to save transcript for user %s: %v", forUser.Email, err)
			}
			
			// Save transcript for "against" user
			err = SaveDebateTranscript(
				againstUser.ID,
				againstUser.Email,
				"user_vs_user",
				topic,
				forUser.Email,
				resultAgainst,
				[]models.Message{}, // You might want to reconstruct messages from transcripts
				againstSubmission.Transcripts,
			)
			if err != nil {
				log.Printf("Failed to save transcript for user %s: %v", againstUser.Email, err)
			}
		} else {
			log.Printf("Error checking for existing transcript: %v", err)
		}
	}

	// Clean up transcripts (optional)
	_, err = transcriptCollection.DeleteMany(ctx, bson.M{"roomId": roomID})
	if err != nil {
		log.Printf("Failed to clean up transcripts: %v", err)
	}

	return map[string]interface{}{
		"message": "Debate judged",
		"result":  result,
	}, nil
	}

	// If only one side has submitted, return a waiting message
	return map[string]interface{}{
		"message": "Waiting for opponent submission",
	}, nil
}

// mergeTranscripts and JudgeDebateHumanVsHuman remain unchanged
func mergeTranscripts(forTranscripts, againstTranscripts map[string]string) map[string]string {
	merged := make(map[string]string)
	for phase, transcript := range forTranscripts {
		merged[phase] = transcript
	}
	for phase, transcript := range againstTranscripts {
		merged[phase] = transcript
	}
	return merged
}

func JudgeDebateHumanVsHuman(merged map[string]string) string {
	if geminiClient == nil {
		log.Println("Gemini client not initialized")
		return "Unable to judge."
	}

	var transcript strings.Builder
	phaseOrder := []string{
		"openingFor", "openingAgainst",
		"crossForQuestion", "crossAgainstAnswer",
		"crossAgainstQuestion", "crossForAnswer",
		"closingFor", "closingAgainst",
	}
	for _, phase := range phaseOrder {
		if text, exists := merged[phase]; exists && text != "" {
			role := "For"
			if strings.Contains(phase, "Against") {
				role = "Against"
			}
			transcript.WriteString(fmt.Sprintf("%s (%s): %s\n", role, phase, text))
		}
	}

	prompt := fmt.Sprintf(
		`Act as a professional debate judge. Analyze the following human-vs-human debate transcript and provide scores in STRICT JSON format:

Judgment Criteria:
1. Opening Statement (10 points):
   - Strength of opening: Clarity of position, persuasiveness
   - Quality of reasoning: Validity, relevance, logical flow
   - Diction/Expression: Language proficiency, articulation

2. Cross Examination Questions (10 points):
   - Validity and relevance to core issues
   - Demonstration of high-order thinking
   - Creativity/Originality ("out-of-the-box" nature)

3. Answers to Cross Examination (10 points):
   - Precision and directness (avoids evasion)
   - Logical coherence
   - Effectiveness in addressing the question

4. Closing Statements (10 points):
   - Comprehensive summary of key points
   - Effective reiteration of stance
   - Persuasiveness of final argument

Required Output Format:
{
  "opening_statement": {
    "for": {"score": X, "reason": "text"},
    "against": {"score": Y, "reason": "text"}
  },
  "cross_examination_questions": {
    "for": {"score": X, "reason": "text"},
    "against": {"score": Y, "reason": "text"}
  },
  "cross_examination_answers": {
    "for": {"score": X, "reason": "text"},
    "against": {"score": Y, "reason": "text"}
  },
  "closing": {
    "for": {"score": X, "reason": "text"},
    "against": {"score": Y, "reason": "text"}
  },
  "total": {
    "for": X,
    "against": Y
  },
  "verdict": {
    "winner": "For/Against",
    "reason": "text",
    "congratulations": "text",
    "opponent_analysis": "text"
  }
}

Debate Transcript:
%s

Provide ONLY the JSON output without any additional text.`, transcript.String())

	ctx := context.Background()
	model := geminiClient.GenerativeModel("gemini-2.5-flash")

	model.SafetySettings = []*genai.SafetySetting{
		{Category: genai.HarmCategoryHarassment, Threshold: genai.HarmBlockNone},
		{Category: genai.HarmCategoryHateSpeech, Threshold: genai.HarmBlockNone},
		{Category: genai.HarmCategorySexuallyExplicit, Threshold: genai.HarmBlockNone},
		{Category: genai.HarmCategoryDangerousContent, Threshold: genai.HarmBlockNone},
	}

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Printf("Gemini error: %v", err)
		return "Unable to judge."
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		if text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
			return string(text)
		}
	}
	return "Unable to judge."
}

// SaveDebateTranscript saves a debate transcript for later viewing
func SaveDebateTranscript(userID primitive.ObjectID, email, debateType, topic, opponent, result string, messages []models.Message, transcripts map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.MongoDatabase.Collection("saved_debate_transcripts")

	// Check if a similar transcript already exists to prevent duplicates
	// Look for transcripts with the same user, topic, opponent, and debate type created within the last 5 minutes
	filter := bson.M{
		"userId":     userID,
		"topic":      topic,
		"opponent":   opponent,
		"debateType": debateType,
		"createdAt":  bson.M{"$gte": time.Now().Add(-5 * time.Minute)},
	}

	var existingTranscript models.SavedDebateTranscript
	err := collection.FindOne(ctx, filter).Decode(&existingTranscript)
	if err == nil {
		// Transcript already exists, check if we need to update it
		log.Printf("Existing transcript found for user %s, topic %s, opponent %s. Current result: %s, New result: %s", 
			email, topic, opponent, existingTranscript.Result, result)
		
		// If the result has changed or is "pending", update the transcript
		if existingTranscript.Result != result || existingTranscript.Result == "pending" {
			update := bson.M{
				"$set": bson.M{
					"result":    result,
					"messages":  messages,
					"transcripts": transcripts,
					"updatedAt": time.Now(),
				},
			}
			
			_, err = collection.UpdateOne(ctx, bson.M{"_id": existingTranscript.ID}, update)
			if err != nil {
				log.Printf("Failed to update existing transcript: %v", err)
				return fmt.Errorf("failed to update transcript: %v", err)
			}
			
			log.Printf("Successfully updated transcript for user %s: %s vs %s, Result: %s -> %s", 
				email, topic, opponent, existingTranscript.Result, result)
			return nil
		} else {
			// Result hasn't changed, skip saving to prevent duplicates
			log.Printf("Transcript already exists with same result (%s), skipping save.", result)
			return nil
		}
	} else if err != mongo.ErrNoDocuments {
		// Error occurred while checking, log it but proceed with saving
		log.Printf("Error checking for existing transcript: %v", err)
	}

	savedTranscript := models.SavedDebateTranscript{
		UserID:      userID,
		Email:       email,
		DebateType:  debateType,
		Topic:       topic,
		Opponent:    opponent,
		Result:      result,
		Messages:    messages,
		Transcripts: transcripts,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = collection.InsertOne(ctx, savedTranscript)
	if err != nil {
		return fmt.Errorf("failed to save transcript: %v", err)
	}

	log.Printf("Successfully saved transcript for user %s: %s vs %s", email, topic, opponent)
	return nil
}

// UpdatePendingTranscripts updates any existing transcripts with "pending" results
func UpdatePendingTranscripts() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collection := db.MongoDatabase.Collection("saved_debate_transcripts")

	// Find all transcripts with "pending" results
	filter := bson.M{"result": "pending"}
	
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to find pending transcripts: %v", err)
	}
	defer cursor.Close(ctx)

	var pendingTranscripts []models.SavedDebateTranscript
	if err = cursor.All(ctx, &pendingTranscripts); err != nil {
		return fmt.Errorf("failed to decode pending transcripts: %v", err)
	}

	log.Printf("Found %d transcripts with pending results", len(pendingTranscripts))

	// Update each pending transcript to have a default result
	for _, transcript := range pendingTranscripts {
		var newResult string
		
		// Determine appropriate result based on debate type
		if transcript.DebateType == "user_vs_bot" {
			// For bot debates, default to loss (assuming bot won)
			newResult = "loss"
		} else {
			// For human debates, default to draw
			newResult = "draw"
		}

		update := bson.M{
			"$set": bson.M{
				"result":    newResult,
				"updatedAt": time.Now(),
			},
		}

		_, err = collection.UpdateOne(ctx, bson.M{"_id": transcript.ID}, update)
		if err != nil {
			log.Printf("Failed to update pending transcript %s: %v", transcript.ID.Hex(), err)
			continue
		}

		log.Printf("Updated pending transcript %s: %s -> %s", transcript.ID.Hex(), transcript.Result, newResult)
	}

	return nil
}

// GetUserDebateTranscripts retrieves all saved debate transcripts for a user
func GetUserDebateTranscripts(userID primitive.ObjectID) ([]models.SavedDebateTranscript, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.MongoDatabase.Collection("saved_debate_transcripts")

	filter := bson.M{"userId": userID}
	opts := &options.FindOptions{
		Sort: bson.M{"createdAt": -1}, // Most recent first
	}

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find transcripts: %v", err)
	}
	defer cursor.Close(ctx)

	var transcripts []models.SavedDebateTranscript
	if err = cursor.All(ctx, &transcripts); err != nil {
		return nil, fmt.Errorf("failed to decode transcripts: %v", err)
	}

	return transcripts, nil
}

// GetDebateTranscriptByID retrieves a specific debate transcript by ID
func GetDebateTranscriptByID(transcriptID primitive.ObjectID, userID primitive.ObjectID) (*models.SavedDebateTranscript, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.MongoDatabase.Collection("saved_debate_transcripts")

	filter := bson.M{"_id": transcriptID, "userId": userID}

	var transcript models.SavedDebateTranscript
	err := collection.FindOne(ctx, filter).Decode(&transcript)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("transcript not found")
		}
		return nil, fmt.Errorf("failed to find transcript: %v", err)
	}

	// If the transcript has a pending result, try to determine the actual result
	if transcript.Result == "pending" {
		log.Printf("Found pending transcript %s, attempting to determine result", transcriptID.Hex())
		
		var newResult string
		
		// Determine appropriate result based on debate type
		if transcript.DebateType == "user_vs_bot" {
			// For bot debates, analyze the messages to determine winner
			newResult = determineBotDebateResult(transcript.Messages)
		} else {
			// For human debates, default to draw
			newResult = "draw"
		}

		// Update the transcript with the determined result
		update := bson.M{
			"$set": bson.M{
				"result":    newResult,
				"updatedAt": time.Now(),
			},
		}

		_, err = collection.UpdateOne(ctx, bson.M{"_id": transcriptID}, update)
		if err != nil {
			log.Printf("Failed to update pending transcript %s: %v", transcriptID.Hex(), err)
		} else {
			log.Printf("Updated pending transcript %s: %s -> %s", transcriptID.Hex(), transcript.Result, newResult)
			transcript.Result = newResult
			transcript.UpdatedAt = time.Now()
		}
	}

	return &transcript, nil
}

// DeleteDebateTranscript deletes a saved debate transcript
func DeleteDebateTranscript(transcriptID primitive.ObjectID, userID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.MongoDatabase.Collection("saved_debate_transcripts")

	filter := bson.M{"_id": transcriptID, "userId": userID}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete transcript: %v", err)
	}

	if result.DeletedCount == 0 {
		return errors.New("transcript not found or not authorized to delete")
	}

	return nil
}

// determineBotDebateResult analyzes the messages from a bot debate to determine the winner
func determineBotDebateResult(messages []models.Message) string {
	// Look for judge messages or final evaluation messages
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		text := strings.ToLower(message.Text)
		
		// Check for judge messages
		if message.Sender == "Judge" {
			if strings.Contains(text, "user win") || strings.Contains(text, "user wins") || 
			   (strings.Contains(text, "user") && strings.Contains(text, "win")) {
				return "win"
			} else if strings.Contains(text, "bot win") || strings.Contains(text, "bot wins") || 
				strings.Contains(text, "lose") || strings.Contains(text, "loss") ||
				(strings.Contains(text, "bot") && strings.Contains(text, "win")) {
				return "loss"
			} else if strings.Contains(text, "draw") {
				return "draw"
			}
		}
		
		// Check for evaluation messages in the last few messages
		if i >= len(messages)-3 {
			if strings.Contains(text, "user win") || strings.Contains(text, "user wins") || 
			   (strings.Contains(text, "user") && strings.Contains(text, "win")) {
				return "win"
			} else if strings.Contains(text, "bot win") || strings.Contains(text, "bot wins") || 
				strings.Contains(text, "lose") || strings.Contains(text, "loss") ||
				(strings.Contains(text, "bot") && strings.Contains(text, "win")) {
				return "loss"
			} else if strings.Contains(text, "draw") {
				return "draw"
			}
		}
	}
	
	// If no clear winner is found, default to loss (assuming bot won)
	return "loss"
}

// GetDebateStats retrieves debate statistics for a user
func GetDebateStats(userID primitive.ObjectID) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.MongoDatabase.Collection("saved_debate_transcripts")

	filter := bson.M{"userId": userID}
	opts := &options.FindOptions{
		Sort: bson.M{"createdAt": -1}, // Most recent first
	}

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find transcripts: %v", err)
	}
	defer cursor.Close(ctx)

	var transcripts []models.SavedDebateTranscript
	if err = cursor.All(ctx, &transcripts); err != nil {
		return nil, fmt.Errorf("failed to decode transcripts: %v", err)
	}

	// Calculate statistics
	totalDebates := len(transcripts)
	wins := 0
	losses := 0
	draws := 0

	// Get recent debates (last 10)
	recentDebates := make([]map[string]interface{}, 0)
	for i, transcript := range transcripts {
		if i >= 10 { // Only get last 10 debates
			break
		}

		// Count results
		switch transcript.Result {
		case "win":
			wins++
		case "loss":
			losses++
		case "draw":
			draws++
		}

		// Add to recent debates
		recentDebates = append(recentDebates, map[string]interface{}{
			 "id":         transcript.ID.Hex(),
			"topic":       transcript.Topic,
			"result":      transcript.Result,
			"opponent":    transcript.Opponent,
			"debateType":  transcript.DebateType,
			"date":        transcript.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"eloChange":   0, // TODO: Add actual Elo change tracking
		})
	}

	winRate := 0.0
	if totalDebates > 0 {
		winRate = float64(wins) / float64(totalDebates) * 100
	}

	return map[string]interface{}{
		"totalDebates":  totalDebates,
		"wins":          wins,
		"losses":        losses,
		"draws":         draws,
		"winRate":       winRate,
		"recentDebates": recentDebates,
	}, nil
}
