package main

import (
	// "context"
	appConfig "arguehub/config"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

// MessageType constants
const (
	MessageTypeDebateStart  = "DEBATE_START"
	MessageTypeDebateEnd    = "DEBATE_END"
	MessageTypeSectionStart = "SECTION_START"
	MessageTypeSectionEnd   = "SECTION_END"
	MessageTypeTurnStart    = "TURN_START"
	MessageTypeTurnEnd      = "TURN_END"
	PingMessage             = "PING"
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
	DebateStarted bool   // Flag to indicate if the debate has started
	CurrentTurn   string // Tracks the current user ID whose turn it is
}

// Global room storage
var (
	rooms  = make(map[string]*Room)
	roomMu sync.Mutex
)

// JSON helper function
func toJSON(data interface{}) string {
	bytes, _ := json.Marshal(data)
	return string(bytes)
}

// Send a WebSocket message
func sendMessage(conn *websocket.Conn, messageType string, data interface{}) error {
	message := Message{
		Type:    messageType,
		Content: toJSON(data),
	}
	if err := conn.WriteJSON(message); err != nil {
		log.Printf("Error sending %s message: %v\n", messageType, err)
		return err
	}
	return nil
}

// Broadcast a message to all users in the room
func broadcastMessage(room *Room, messageType string, data interface{}) {
	room.Mutex.Lock()
	defer room.Mutex.Unlock()
	for userID, conn := range room.Users {
		log.Printf("Sending %s to user: %s", messageType, userID)
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
			// Close the old connection
			existingConn.Close()
			// Update with the new connection
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

	// Create a new room
	newRoom := &Room{
		Users:     map[string]*websocket.Conn{userID: conn},
		DebateFmt: getDebateFormat(),
	}
	roomID := generateRoomID()
	rooms[roomID] = newRoom

	// Connection verification
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
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Println("Error upgrading WebSocket:", err)
		return
	}
	defer conn.Close()

	queryParams := ctx.Request.URL.Query()["userId"]
	userID := queryParams[0]
	log.Printf("WebSocket connection established for userId: %s", userID)

	room, err := createOrJoinRoom(userID, conn)
	if err != nil {
		log.Println("Error joining room:", err)
		return
	}

	// Wait for a second user to join
	log.Println("Waiting for another user...")
	for {
		room.Mutex.Lock()
		if len(room.Users) == 2 && !room.DebateStarted {
			room.DebateStarted = true // Set the flag to true
			room.Mutex.Unlock()
			break
		}
		room.Mutex.Unlock()
		time.Sleep(1 * time.Second)
	}

	log.Println("Two users connected. Starting debate.")
	startDebate(room)

	// Close all connections and expire the room after the debate ends
	closeConnectionsAndExpireRoom(room)
}

// Start the debate
func startDebate(room *Room) {
	broadcastMessage(room, MessageTypeDebateStart, nil)

	for _, section := range room.DebateFmt.Sections {
		broadcastMessage(room, MessageTypeSectionStart, CurrentStatus{Section: section.Name})

		for userID, _ := range room.Users {
			room.Mutex.Lock()
			room.CurrentTurn = userID
			room.Mutex.Unlock()

			turnStatus := CurrentStatus{
				CurrentTurn: userID,
				Section:     section.Name,
				Duration:    int(section.Duration.Seconds()),
			}
			broadcastMessage(room, MessageTypeTurnStart, turnStatus)

			time.Sleep(section.Duration)

			broadcastMessage(room, MessageTypeTurnEnd, nil)
		}

		broadcastMessage(room, MessageTypeSectionEnd, nil)
	}

	broadcastMessage(room, MessageTypeDebateEnd, nil)
}

// Handle media data sent during a user's turn
func handleMediaData(room *Room, userID string, data []byte, section string) error {
	room.Mutex.Lock()
	defer room.Mutex.Unlock()

	// Validate the sender's turn
	if room.CurrentTurn != userID {
		return fmt.Errorf("not your turn")
	}

	// Save the media data to the folder ./temp
	filename := fmt.Sprintf("./temp/%s_%s", section, userID)
	if err := os.WriteFile(filename, data, 0644); err != nil {
		log.Printf("Error saving media data for user %s: %v", userID, err)
		return err
	}

	log.Printf("Media data from user %s saved as %s", userID, filename)
	return nil
}

// Generate a unique room ID
func generateRoomID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// Initialize debate format
func getDebateFormat() DebateFormat {
	return DebateFormat{
		Sections: []Section{
			{Name: "Opening", Duration: 15 * time.Second},
			{Name: "Rebuttal", Duration: 20 * time.Second},
			{Name: "Closing", Duration: 10 * time.Second},
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

	// Remove the room from the global storage
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
