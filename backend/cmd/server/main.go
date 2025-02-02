package main

import (
	// "context"
	appConfig "arguehub/config"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	// "io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	connectedUsers = make(map[string]bool)
	mu             sync.Mutex // Mutex to protect connectedUsers
)

func SignUpRouteHandler(ctx *gin.Context) {
	rootPath, _ := os.Getwd()
	if rootPath == "" {
		log.Println("rootpath is not set")
		return
	}

	configPath := filepath.Join(rootPath, "config", "config.prod.yml")
	cfg, err := appConfig.LoadConfig(configPath)
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		ctx.JSON(500, gin.H{"error": "Failed to load config"})
		return
	}

	fmt.Println("signing up")

	var request struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(400, gin.H{"error": "Invalid input", "message": err.Error()})
		return
	}

	fmt.Println(cfg.Cognito.AppClientId, cfg.Cognito.AppClientSecret)
	err = signUpWithCognito(cfg.Cognito.AppClientId, cfg.Cognito.AppClientSecret, request.Email, request.Password, ctx)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "Failed to sign up", "message": err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"message": "Sign-up successful"})
}

func signUpWithCognito(appClientId, appClientSecret, email, password string, ctx *gin.Context) error {
	config, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion("ap-south-1"))
	if err != nil {
		log.Println("Error loading AWS config:", err)
		return fmt.Errorf("failed to load AWS config: %v", err)
	}

	cognitoClient := cognitoidentityprovider.NewFromConfig(config)

	secretHash := generateSecretHash(email, appClientId, appClientSecret)

	signupInput := cognitoidentityprovider.SignUpInput{
		ClientId:   aws.String(appClientId),
		Password:   aws.String(password),
		SecretHash: aws.String(secretHash),
		Username:   aws.String(email),
		UserAttributes: []types.AttributeType{
			{
				Name:  aws.String("email"),
				Value: aws.String(email),
			},
			{
				Name:  aws.String("nickname"),
				Value: aws.String(extractNameFromEmail(email)),
			},
		},
	}

	signupStatus, err := cognitoClient.SignUp(ctx, &signupInput)
	if err != nil {
		log.Println("Error during sign-up:", err)
		return fmt.Errorf("sign-up failed: %v", err)
	}

	log.Println("Sign-up successful:", signupStatus)
	return nil
}

func VerifyEmailRouteHandler(ctx *gin.Context) {
	rootPath, _ := os.Getwd()
	if rootPath == "" {
		log.Println("rootpath is not set")
		return
	}

	configPath := filepath.Join(rootPath, "config", "config.prod.yml")
	cfg, err := appConfig.LoadConfig(configPath)
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		ctx.JSON(500, gin.H{"error": "Failed to load config"})
		return
	}

	var request struct {
		Email            string `json:"email" binding:"required,email"`
		ConfirmationCode string `json:"confirmationCode" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(400, gin.H{"error": "Invalid input", "message": err.Error()})
		return
	}

	err = verifyEmailWithCognito(cfg.Cognito.AppClientId, cfg.Cognito.AppClientSecret, request.Email, request.ConfirmationCode, ctx)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "Failed to verify email", "message": err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"message": "Email verification successful"})
}

func verifyEmailWithCognito(appClientId, appClientSecret, email, confirmationCode string, ctx *gin.Context) error {
	config, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion("ap-south-1"))
	if err != nil {
		log.Println("Error loading AWS config:", err)
		return fmt.Errorf("failed to load AWS config: %v", err)
	}

	cognitoClient := cognitoidentityprovider.NewFromConfig(config)

	secretHash := generateSecretHash(email, appClientId, appClientSecret)

	confirmSignUpInput := cognitoidentityprovider.ConfirmSignUpInput{
		ClientId:         aws.String(appClientId),
		ConfirmationCode: aws.String(confirmationCode),
		Username:         aws.String(email),
		SecretHash:       aws.String(secretHash),
	}

	confirmationStatus, err := cognitoClient.ConfirmSignUp(ctx, &confirmSignUpInput)
	if err != nil {
		log.Println("Error during email verification:", err)
		return fmt.Errorf("email verification failed: %v", err)
	}

	log.Println("Email verification successful:", confirmationStatus)
	return nil
}

func generateSecretHash(username, clientId, clientSecret string) string {
	hmacInstance := hmac.New(sha256.New, []byte(clientSecret))
	hmacInstance.Write([]byte(username + clientId))
	secretHashByte := hmacInstance.Sum(nil)

	secretHashString := base64.StdEncoding.EncodeToString(secretHashByte)
	return secretHashString
}

func extractNameFromEmail(email string) string {
	re := regexp.MustCompile(`^([^@]+)`)

	match := re.FindStringSubmatch(email)

	return match[1]
}

func LoginRouteHandler(ctx *gin.Context) {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "./config/config.prod.yml"
	}
	cfg, err := appConfig.LoadConfig(cfgPath)
	if err != nil {
		log.Println("Failed to load config")
		ctx.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	var request struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(400, gin.H{"error": "Invalid input", "message": "Check email and password format"})
		return
	}

	token, err := loginWithCognito(cfg.Cognito.AppClientId, cfg.Cognito.AppClientSecret, request.Email, request.Password, ctx)
	if err != nil {
		ctx.JSON(401, gin.H{"error": "Failed to sign in", "message": "Invalid email or password"})
		return
	}

	ctx.JSON(200, gin.H{"message": "Sign-in successful", "accessToken": token})
}

func loginWithCognito(appClientId, appClientSecret, email, password string, ctx *gin.Context) (string, error) {
	config, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion("ap-south-1"))
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config")
	}

	cognitoClient := cognitoidentityprovider.NewFromConfig(config)
	secretHash := generateSecretHash(email, appClientId, appClientSecret)

	authInput := cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeUserPasswordAuth,
		ClientId: aws.String(appClientId),
		AuthParameters: map[string]string{
			"USERNAME":    email,
			"PASSWORD":    password,
			"SECRET_HASH": secretHash,
		},
	}

	authOutput, err := cognitoClient.InitiateAuth(ctx, &authInput)
	if err != nil {
		return "", fmt.Errorf("authentication failed")
	}

	return *authOutput.AuthenticationResult.AccessToken, nil
}

func ForgotPasswordRouteHandler(ctx *gin.Context) {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "./config/config.prod.yml"
	}
	cfg, err := appConfig.LoadConfig(cfgPath)
	if err != nil {
		log.Println("Failed to load config")
		ctx.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	var request struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(400, gin.H{"error": "Invalid input", "message": "Check email format"})
		return
	}

	_, err = initiateForgotPassword(cfg.Cognito.AppClientId, cfg.Cognito.AppClientSecret, request.Email, ctx)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "Failed to initiate password reset", "message": err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"message": "Password reset initiated. Check your email for further instructions."})
}

func initiateForgotPassword(appClientId, appClientSecret, email string, ctx *gin.Context) (*cognitoidentityprovider.ForgotPasswordOutput, error) {
	config, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion("ap-south-1"))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config")
	}

	cognitoClient := cognitoidentityprovider.NewFromConfig(config)
	secretHash := generateSecretHash(email, appClientId, appClientSecret)

	forgotPasswordInput := cognitoidentityprovider.ForgotPasswordInput{
		ClientId:   aws.String(appClientId),
		Username:   aws.String(email),
		SecretHash: aws.String(secretHash),
	}

	output, err := cognitoClient.ForgotPassword(ctx, &forgotPasswordInput)
	if err != nil {
		return nil, fmt.Errorf("error initiating forgot password: %v", err)
	}

	return output, nil
}

func VerifyForgotPasswordRouteHandler(ctx *gin.Context) {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "./config/config.prod.yml"
	}
	cfg, err := appConfig.LoadConfig(cfgPath)
	if err != nil {
		log.Println("Failed to load config")
		ctx.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	var request struct {
		Email       string `json:"email" binding:"required,email"`
		Code        string `json:"code" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required,min=8"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(400, gin.H{"error": "Invalid input", "message": err.Error()})
		return
	}

	_, err = confirmForgotPassword(cfg.Cognito.AppClientId, cfg.Cognito.AppClientSecret, request.Email, request.Code, request.NewPassword, ctx)
	if err != nil {
		ctx.JSON(500, gin.H{"error": "Failed to confirm password reset", "message": err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"message": "Password successfully changed"})
}

func confirmForgotPassword(appClientId, appClientSecret, email, code, newPassword string, ctx *gin.Context) (*cognitoidentityprovider.ConfirmForgotPasswordOutput, error) {
	config, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion("ap-south-1"))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config")
	}

	cognitoClient := cognitoidentityprovider.NewFromConfig(config)
	secretHash := generateSecretHash(email, appClientId, appClientSecret)

	confirmForgotPasswordInput := cognitoidentityprovider.ConfirmForgotPasswordInput{
		ClientId:         aws.String(appClientId),
		Username:         aws.String(email),
		ConfirmationCode: aws.String(code),
		Password:         aws.String(newPassword),
		SecretHash:       aws.String(secretHash),
	}

	output, err := cognitoClient.ConfirmForgotPassword(ctx, &confirmForgotPasswordInput)
	if err != nil {
		return nil, fmt.Errorf("error confirming forgot password: %v", err)
	}

	return output, nil
}

func VerifyTokenRouteHandler(ctx *gin.Context) {
	// Get the Authorization header
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		ctx.JSON(401, gin.H{"error": "Missing token"})
		return
	}

	// Extract token (expected format: "Bearer <token>")
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		ctx.JSON(400, gin.H{"error": "Invalid token format"})
		return
	}
	token := tokenParts[1]

	// Load AWS Cognito configuration
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "./config/config.prod.yml"
	}
	cfg, err := appConfig.LoadConfig(cfgPath)
	if err != nil {
		log.Println("Failed to load config")
		ctx.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	// Validate the token
	valid, err := validateTokenWithCognito(cfg.Cognito.UserPoolId, token, ctx)
	if err != nil {
		ctx.JSON(401, gin.H{"error": "Invalid or expired token", "message": err.Error()})
		return
	}

	if !valid {
		ctx.JSON(401, gin.H{"error": "Token is invalid or expired"})
		return
	}

	ctx.JSON(200, gin.H{"message": "Token is valid"})
}

func validateTokenWithCognito(userPoolId, token string, ctx *gin.Context) (bool, error) {
	config, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion("ap-south-1"))
	if err != nil {
		return false, fmt.Errorf("failed to load AWS config")
	}

	cognitoClient := cognitoidentityprovider.NewFromConfig(config)

	// Call Cognito to verify the token
	_, err = cognitoClient.GetUser(ctx, &cognitoidentityprovider.GetUserInput{
		AccessToken: aws.String(token),
	})
	if err != nil {
		log.Println("Token verification failed:", err)
		return false, fmt.Errorf("token validation failed: %v", err)
	}

	return true, nil
}


// Constants for message types
const (
	MessageTypeDebateStart          = "DEBATE_START"
	MessageTypeDebateEnd            = "DEBATE_END"
	MessageTypeSectionStart         = "SECTION_START"
	MessageTypeSectionEnd           = "SECTION_END"
	MessageTypeTurnStart            = "TURN_START"
	MessageTypeTurnEnd              = "TURN_END"
	MessageTypeGeneratingTranscript = "GENERATING_TRANSCRIPT"
	MessageTypeChatMessage          = "CHAT_MESSAGE"
	PingMessage                     = "PING"

	ReadBufferSize  = 131022
	WriteBufferSize = 131022
)

// Message represents a message structure sent over WebSocket
type Message struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
}

// CurrentStatus represents the current status of the debate
type CurrentStatus struct {
	CurrentTurn string `json:"currentTurn,omitempty"`
	Section     string `json:"section"`
	Duration    int    `json:"duration,omitempty"` // Duration in seconds
}

// Section defines a debate section with name and duration per turn
type Section struct {
	Name     string
	Duration time.Duration // Duration per turn in this section
}

// DebateFormat defines the structure and timing of the debate
type DebateFormat struct {
	Sections []Section `json:"sections"`
}

// Room represents a room with connected users
type Room struct {
	Users         map[string]*websocket.Conn
	Mutex         sync.Mutex
	DebateFmt     DebateFormat
	DebateStarted bool            // Flag to indicate if the debate has started
	CurrentTurn   string          // Tracks the current user ID whose turn it is
	TurnActive    map[string]bool // Tracks whether each user is actively participating in their turn
}

// Global room storage
var (
	rooms  = make(map[string]*Room)
	roomMu sync.Mutex
)

// JSON helper function
func toJSON(data interface{}) (string, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Send a WebSocket message
func sendMessage(conn *websocket.Conn, messageType string, data interface{}) error {
	content, err := toJSON(data)
	if err != nil {
		return fmt.Errorf("error marshaling data: %w", err)
	}

	message := Message{
		Type:    messageType,
		Content: content,
	}
	if err := conn.WriteJSON(message); err != nil {
		return fmt.Errorf("error sending %s message: %w", messageType, err)
	}
	return nil
}

// Broadcast a message to all users in the room
func broadcastMessage(room *Room, messageType string, data interface{}) {
	room.Mutex.Lock()
	defer room.Mutex.Unlock()
	for userID, conn := range room.Users {
		if err := sendMessage(conn, messageType, data); err != nil {
			log.Printf("Error broadcasting to user %s: %v", userID, err)
			conn.Close()
			delete(room.Users, userID)
		}
	}
}

// Create or join a room
func createOrJoinRoom(userID string, conn *websocket.Conn) (*Room, error) {
	roomMu.Lock()
	defer roomMu.Unlock()

	for _, room := range rooms {
		room.Mutex.Lock()
		if existingConn, exists := room.Users[userID]; exists {
			existingConn.Close()
			room.Users[userID] = conn
			room.Mutex.Unlock()
			return room, nil
		}
		if len(room.Users) < 2 {
			room.Users[userID] = conn
			room.Mutex.Unlock()
			return room, nil
		}
		room.Mutex.Unlock()
	}

	// Initialize the room with TurnActive map
	newRoom := &Room{
		Users:      map[string]*websocket.Conn{userID: conn},
		DebateFmt:  getDebateFormat(),
		TurnActive: make(map[string]bool), // Initialize TurnActive for each user
	}
	roomID := generateRoomID()
	rooms[roomID] = newRoom

	// Verify connections for this new room
	go verifyConnections(newRoom)

	return newRoom, nil
}

// Verify active connections
func verifyConnections(room *Room) {
	time.Sleep(2 * time.Second)
	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	for userID, conn := range room.Users {
		if err := sendMessage(conn, PingMessage, nil); err != nil {
			log.Printf("Connection lost for user %s, removing from room", userID)
			conn.Close()
			delete(room.Users, userID)
		}
	}
}

// WebSocket handler
func WebsocketHandler(ctx *gin.Context) {
	upgrader := websocket.Upgrader{
		CheckOrigin:       func(r *http.Request) bool { return true },
		ReadBufferSize:    ReadBufferSize,
		WriteBufferSize:   WriteBufferSize,
		EnableCompression: false,
	}

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Println("Error upgrading WebSocket:", err)
		return
	}
	defer conn.Close()

	userID := ctx.Query("userId")
	if userID == "" {
		log.Println("Missing userId in query parameters")
		return
	}

	log.Printf("WebSocket connection established for userId: %s", userID)

	room, err := createOrJoinRoom(userID, conn)
	if err != nil {
		log.Println("Error joining room:", err)
		return
	}

	log.Println("Waiting for another user to join...")
	for {
		room.Mutex.Lock()
		if len(room.Users) == 2 && !room.DebateStarted {
			room.DebateStarted = true
			room.Mutex.Unlock()
			break
		}
		room.Mutex.Unlock()
		time.Sleep(1 * time.Second)
	}

	log.Println("Two users connected. Starting debate.")

	startDebate(room)

	closeConnectionsAndExpireRoom(room)
}

type ChatMessage struct {
	Sender  string `json:"sender"`
	Message string `json:"message"`
}

type GameResult struct {
	WinnerUserId string `json:"winnerUserId"`
	Points int `json:"points"`
	TotalPoints int `json:"totalPoints"`
	EvaluationMessage string `json:"evaluationMessage"`
}

func startDebate(room *Room) {
	broadcastMessage(room, MessageTypeDebateStart, nil)

	for _, section := range room.DebateFmt.Sections {
		log.Printf("Section: %s", section.Name)
		broadcastMessage(room, MessageTypeSectionStart, CurrentStatus{Section: section.Name})

		for userID, conn := range room.Users {
			room.Mutex.Lock()
			room.CurrentTurn = userID
			room.Mutex.Unlock()

			turnStatus := CurrentStatus{
				CurrentTurn: userID,
				Section:     section.Name,
				Duration:    int(section.Duration.Seconds()),
			}

			// Mark the user's turn as active
			room.Mutex.Lock()
			room.TurnActive[userID] = true
			room.Mutex.Unlock()

			time.Sleep(time.Second * 2)
			broadcastMessage(room, MessageTypeTurnStart, turnStatus)

			// Save user media
			mediaFileChan := make(chan string)
			go saveUserMedia(conn, userID, section.Name, mediaFileChan, room)

			time.Sleep(section.Duration)
			// End current turn
			broadcastMessage(room, MessageTypeTurnEnd, nil)

			// Mark the user's turn as inactive
			room.Mutex.Lock()
			room.TurnActive[userID] = false
			room.Mutex.Unlock()

			// Wait for media file path
			mediaFilePath := <-mediaFileChan
			if mediaFilePath != "" {
				// Generate transcript
				// Notify frontend that transcript is being generated
				broadcastMessage(room, MessageTypeGeneratingTranscript, ChatMessage{
					Sender:  userID,
					Message: "Transcript is being generated...",
				})

				transcript, err := generateTranscript(mediaFilePath)
				if err != nil {
					log.Printf("Error generating transcript for user %s: %v", userID, err)
					continue
				}

				// Broadcast transcript as a chat message
				broadcastMessage(room, MessageTypeChatMessage, ChatMessage{
					Sender:  userID,
					Message: transcript,
				})
			}
		}

		broadcastMessage(room, MessageTypeSectionEnd, nil)
	}

	broadcastMessage(room, MessageTypeDebateEnd, nil)

	broadcastMessage(room, "GENERATING_RESULTS", nil);

	gameResult := GameResult{
		WinnerUserId: "1",
		Points: 10,
		TotalPoints: 100,
		EvaluationMessage: "you won the match, for the reasons you don't need to know",
	}
	broadcastMessage(room, "GAME_RESULT", gameResult);
}

func saveUserMedia(conn *websocket.Conn, userID, sectionName string, mediaFileChan chan<- string, room *Room) {
	defer close(mediaFileChan)

	tempFilename := fmt.Sprintf("temp_media_%s_%s.webm", userID, sectionName)
	finalFilename := fmt.Sprintf("media_%s_%s.webm", userID, sectionName)

	file, err := os.Create(tempFilename)
	if err != nil {
		log.Printf("Error creating file for user %s: %v", userID, err)
		mediaFileChan <- ""
		return
	}
	defer func() {
		file.Close()
		err = os.Rename(tempFilename, finalFilename)
		if err != nil {
			log.Printf("Error renaming file for user %s: %v", userID, err)
			mediaFileChan <- ""
		} else {
			log.Printf("Media saved for user %s in section %s", userID, sectionName)
			mediaFileChan <- finalFilename
		}
	}()

	for {
		room.Mutex.Lock()
		active := room.TurnActive[userID]
		room.Mutex.Unlock()

		if !active {
			log.Printf("Turn ended for user %s. Stopping media collection.", userID)
			break
		}

		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("Connection closed for user %s", userID)
			} else {
				log.Printf("Error reading chunk for user %s: %v", userID, err)
			}
			break
		}

		if messageType == websocket.BinaryMessage {
			_, err = file.Write(data)
			if err != nil {
				log.Printf("Error writing chunk for user %s: %v", userID, err)
				break
			}
		}
	}

	err = file.Sync()
	if err != nil {
		log.Printf("Error syncing file for user %s: %v", userID, err)
		mediaFileChan <- ""
	}
}

type TranscriptionResponse struct {
	Transcription string `json:"transcription"`
	Error         string `json:"error"`
}
func generateTranscript(mediaFilePath string) (string, error) {
	serverURL := "http://localhost:8000/transcribe/batch"

	payload := map[string]string{"file_path": mediaFilePath}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON payload: %v", err)
	}

	resp, err := http.Post(serverURL, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	var result TranscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("server error: %s", result.Error)
	}

	return result.Transcription, nil
}

// Generate a unique room ID
func generateRoomID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// Initialize debate format
func getDebateFormat() DebateFormat {
	return DebateFormat{
		Sections: []Section{
			{Name: "Opening", Duration: 2 * time.Second},
			// {Name: "Rebuttal", Duration: 3 * time.Second},
			// {Name: "Closing", Duration: 3 * time.Second},
		},
	}
}

func closeConnectionsAndExpireRoom(room *Room) {
	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	for userID, conn := range room.Users {
		log.Printf("Closing connection for user: %s", userID)
		conn.Close()
		delete(room.Users, userID)
	}

	roomMu.Lock()
	defer roomMu.Unlock()
	for roomID, r := range rooms {
		if r == room {
			delete(rooms, roomID)
			log.Printf("Room %s expired and removed", roomID)
			break
		}
	}
}

// TranscriptionResult represents the JSON response from the Python script
type TranscriptionResult struct {
	Transcription string `json:"transcription"`
}

func server() {
	// gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	router.SetTrustedProxies([]string{"127.0.0.1", "localhost"}) // Adjust this for production if needed

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	router.OPTIONS("/*path", func(c *gin.Context) {
		c.Status(204)
	})
	router.POST("/signup", SignUpRouteHandler)
	router.POST("/verifyEmail", VerifyEmailRouteHandler)
	router.POST("/login", LoginRouteHandler)
	router.POST("/forgotPassword", ForgotPasswordRouteHandler)
	router.POST("/confirmForgotPassword", VerifyForgotPasswordRouteHandler)
	router.POST("/verifyToken", VerifyTokenRouteHandler)
	
	router.GET("/ws", WebsocketHandler)

	rootPath, _ := os.Getwd()
	if rootPath == "" {
		log.Println("rootpath is not set")
		return
	}
	configPath := filepath.Join(rootPath, "config", "config.prod.yml")
	cfg, err := appConfig.LoadConfig(configPath)
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		return
	}

	port := strconv.Itoa(cfg.Server.Port)
	router.Run(":" + port)
}

func main() {
	server()
}
