package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"arguehub/db"
	"arguehub/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
)

var upgrader = websocket.Upgrader{
	// In production, adjust the CheckOrigin function to allow only trusted origins.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Room represents a debate room with connected clients.
type Room struct {
	Clients map[*websocket.Conn]*Client
	Mutex   sync.Mutex
}

// Client represents a connected client with user information
type Client struct {
	Conn     *websocket.Conn
	UserID   string
	Username string
	Email    string
	IsTyping bool
	IsSpeaking bool
	PartialText string
	LastActivity time.Time
	IsMuted bool // New field to track mute status
	Role string // New field to track debate role (for/against)
	SpeechText string // New field to store speech text
}

type Message struct {
	Type     string          `json:"type"`
	Room     string          `json:"room,omitempty"`
	Username string          `json:"username,omitempty"`
	UserID   string          `json:"userId,omitempty"`
	Content  string          `json:"content,omitempty"`
	Extra    json.RawMessage `json:"extra,omitempty"`
	// New fields for real-time communication
	IsTyping    bool   `json:"isTyping,omitempty"`
	IsSpeaking  bool   `json:"isSpeaking,omitempty"`
	PartialText string `json:"partialText,omitempty"`
	Timestamp   int64  `json:"timestamp,omitempty"`
	Mode        string `json:"mode,omitempty"` // 'type' or 'speak'
	// Debate-specific fields
	Phase string `json:"phase,omitempty"`
	Topic string `json:"topic,omitempty"`
	Role  string `json:"role,omitempty"`
	Ready *bool  `json:"ready,omitempty"`
	// New fields for automatic muting
	IsMuted bool `json:"isMuted,omitempty"`
	CurrentTurn string `json:"currentTurn,omitempty"` // "for" or "against"
	SpeechText string `json:"speechText,omitempty"` // Converted speech to text
}

type TypingIndicator struct {
	UserID      string `json:"userId"`
	Username    string `json:"username"`
	IsTyping    bool   `json:"isTyping"`
	IsSpeaking  bool   `json:"isSpeaking"`
	PartialText string `json:"partialText,omitempty"`
}

var rooms = make(map[string]*Room)
var roomsMutex sync.Mutex

// WebsocketHandler handles WebSocket connections for debate signaling.
func WebsocketHandler(c *gin.Context) {
	// Get token from query parameter
	token := c.Query("token")
	if token == "" {
		log.Println("WebSocket connection failed: missing token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		return
	}

	// Validate token
	valid, email, err := utils.ValidateTokenAndFetchEmail("./config/config.prod.yml", token, c)
	if err != nil || !valid || email == "" {
		log.Printf("WebSocket connection failed: invalid token - %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	roomID := c.Query("room")
	if roomID == "" {
		log.Println("WebSocket connection failed: missing room parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing room parameter"})
		return
	}

	// Get user details from database
	userID, username, err := getUserDetails(email)
	if err != nil {
		log.Printf("Failed to get user details: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user details"})
		return
	}

	// Create the room if it doesn't exist.
	roomsMutex.Lock()
	if _, exists := rooms[roomID]; !exists {
		rooms[roomID] = &Room{Clients: make(map[*websocket.Conn]*Client)}
		log.Printf("Created new room: %s", roomID)
	}
	room := rooms[roomID]
	roomsMutex.Unlock()

	// Upgrade the connection.
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	// Limit room to 2 clients.
	room.Mutex.Lock()
	if len(room.Clients) >= 2 {
		log.Printf("Room %s is full. Closing connection.", roomID)
		room.Mutex.Unlock()
		conn.Close()
		return
	}
	
	// Create client instance
	client := &Client{
		Conn:         conn,
		UserID:       userID,
		Username:     username,
		Email:        email,
		IsTyping:     false,
		IsSpeaking:   false,
		PartialText:  "",
		LastActivity: time.Now(),
		IsMuted:      false, // Initialize mute status
		Role:         "",   // Initialize role
		SpeechText:   "",   // Initialize speech text
	}
	
	room.Clients[conn] = client
	log.Printf("Client %s joined room %s (total clients: %d)", username, roomID, len(room.Clients))
	room.Mutex.Unlock()

	// Listen for messages.
	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error in room %s: %v", roomID, err)
			// Remove client from room.
			room.Mutex.Lock()
			delete(room.Clients, conn)
			log.Printf("Client removed from room %s (total clients: %d)", roomID, len(room.Clients))
			// If room is empty, delete it.
			if len(room.Clients) == 0 {
				roomsMutex.Lock()
				delete(rooms, roomID)
				roomsMutex.Unlock()
				log.Printf("Room %s deleted as it became empty", roomID)
			}
			room.Mutex.Unlock()
			break
		}

		log.Printf("Received message in room %s: %s", roomID, string(msg))

		// Parse the message
		var message Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}
		
		log.Printf("Parsed message type: %s, topic: %s, role: %s, ready: %v", 
			message.Type, message.Topic, message.Role, message.Ready)

		// Update client activity
		room.Mutex.Lock()
		if client, exists := room.Clients[conn]; exists {
			client.LastActivity = time.Now()
		}
		room.Mutex.Unlock()

		// Handle different message types
		switch message.Type {
		case "join":
			// Handle join message - just acknowledge it
			log.Printf("Client %s joined room %s", client.Username, roomID)
		case "message":
			log.Printf("Handling chat message from %s in room %s", client.Username, roomID)
			handleChatMessage(room, conn, message, client)
		case "typing":
			log.Printf("Handling typing indicator from %s in room %s", client.Username, roomID)
			handleTypingIndicator(room, conn, message, client)
		case "speaking":
			log.Printf("Handling speaking indicator from %s in room %s", client.Username, roomID)
			handleSpeakingIndicator(room, conn, message, client)
		case "speechText":
			log.Printf("Handling speech-to-text from %s in room %s", client.Username, roomID)
			handleSpeechText(room, conn, message, client)
		case "phaseChange":
			log.Printf("Handling phase change from %s in room %s: %s", client.Username, roomID, message.Phase)
			handlePhaseChange(room, conn, message)
		case "topicChange":
			log.Printf("Handling topic change from %s in room %s: %s", client.Username, roomID, message.Topic)
			handleTopicChange(room, conn, message)
		case "roleSelection":
			log.Printf("Handling role selection from %s in room %s: %s", client.Username, roomID, message.Role)
			handleRoleSelection(room, conn, message)
		case "ready":
			readyStatus := "false"
			if message.Ready != nil {
				readyStatus = fmt.Sprintf("%v", *message.Ready)
			}
			log.Printf("Handling ready status from %s in room %s: %s", client.Username, roomID, readyStatus)
			handleReadyStatus(room, conn, message)
		case "mute":
			log.Printf("Handling mute request from %s in room %s", client.Username, roomID)
			handleMuteRequest(room, conn, message, client)
		case "unmute":
			log.Printf("Handling unmute request from %s in room %s", client.Username, roomID)
			handleUnmuteRequest(room, conn, message, client)
		default:
			log.Printf("Unknown message type '%s' from %s in room %s, forwarding to other clients", message.Type, client.Username, roomID)
			// Broadcast the message to all other clients in the room.
			room.Mutex.Lock()
			for clientConn := range room.Clients {
				if clientConn != conn {
					if err := clientConn.WriteMessage(messageType, msg); err != nil {
						log.Printf("WebSocket write error in room %s: %v", roomID, err)
					} else {
						log.Printf("Forwarded message to a client in room %s", roomID)
					}
				}
			}
			room.Mutex.Unlock()
		}
	}

	log.Printf("Connection closed in room %s", roomID)
}

// handleChatMessage handles chat messages with enhanced features
func handleChatMessage(room *Room, conn *websocket.Conn, message Message, client *Client) {
	// Add timestamp if not provided
	if message.Timestamp == 0 {
		message.Timestamp = time.Now().Unix()
	}

	// Reset typing/speaking indicators
	room.Mutex.Lock()
	client.IsTyping = false
	client.IsSpeaking = false
	client.PartialText = ""
	room.Mutex.Unlock()

	// Broadcast to other clients
	room.Mutex.Lock()
	for clientConn := range room.Clients {
		if clientConn != conn {
			response := map[string]interface{}{
				"type":      "message",
				"userId":    client.UserID,
				"username":  client.Username,
				"content":   message.Content,
				"timestamp": message.Timestamp,
				"mode":      message.Mode,
			}
			
			if err := clientConn.WriteJSON(response); err != nil {
				log.Printf("WebSocket write error: %v", err)
			}
		}
	}
	room.Mutex.Unlock()
}

// handleTypingIndicator handles typing indicators
func handleTypingIndicator(room *Room, conn *websocket.Conn, message Message, client *Client) {
	room.Mutex.Lock()
	client.IsTyping = message.IsTyping
	client.PartialText = message.PartialText
	room.Mutex.Unlock()

	// Broadcast typing indicator to other clients
	room.Mutex.Lock()
	for clientConn := range room.Clients {
		if clientConn != conn {
			response := map[string]interface{}{
				"type":        "typingIndicator",
				"userId":      client.UserID,
				"username":    client.Username,
				"isTyping":    message.IsTyping,
				"partialText": message.PartialText,
			}
			
			if err := clientConn.WriteJSON(response); err != nil {
				log.Printf("WebSocket write error: %v", err)
			}
		}
	}
	room.Mutex.Unlock()
}

// handleSpeakingIndicator handles speaking indicators
func handleSpeakingIndicator(room *Room, conn *websocket.Conn, message Message, client *Client) {
	room.Mutex.Lock()
	client.IsSpeaking = message.IsSpeaking
	room.Mutex.Unlock()

	// Broadcast speaking indicator to other clients
	room.Mutex.Lock()
	for clientConn := range room.Clients {
		if clientConn != conn {
			response := map[string]interface{}{
				"type":       "speakingIndicator",
				"userId":     client.UserID,
				"username":   client.Username,
				"isSpeaking": message.IsSpeaking,
			}
			
			if err := clientConn.WriteJSON(response); err != nil {
				log.Printf("WebSocket write error: %v", err)
			}
		}
	}
	room.Mutex.Unlock()
}

// handleSpeechText handles speech-to-text conversion
func handleSpeechText(room *Room, conn *websocket.Conn, message Message, client *Client) {
	log.Printf("Received speech text from %s: %s", client.Username, message.SpeechText)
	
	room.Mutex.Lock()
	client.SpeechText = message.SpeechText
	room.Mutex.Unlock()

	// Broadcast speech text to other clients
	room.Mutex.Lock()
	for clientConn := range room.Clients {
		if clientConn != conn {
			response := map[string]interface{}{
				"type": "speechText",
				"userId": client.UserID,
				"username": client.Username,
				"speechText": client.SpeechText,
			}
			
			log.Printf("Broadcasting speech text to other client: %s", client.SpeechText)
			if err := clientConn.WriteJSON(response); err != nil {
				log.Printf("WebSocket write error: %v", err)
			}
		}
	}
	room.Mutex.Unlock()
}

// handlePhaseChange handles phase changes
func handlePhaseChange(room *Room, conn *websocket.Conn, message Message) {
	// Determine whose turn it is based on the phase
	var currentTurn string
	switch message.Phase {
	case "openingFor", "crossForQuestion", "crossForAnswer", "closingFor":
		currentTurn = "for"
	case "openingAgainst", "crossAgainstQuestion", "crossAgainstAnswer", "closingAgainst":
		currentTurn = "against"
	default:
		currentTurn = ""
	}

	log.Printf("Phase change to %s, current turn: %s", message.Phase, currentTurn)

	// Automatically mute/unmute users based on turn
	room.Mutex.Lock()
	for clientConn, client := range room.Clients {
		if client.Role != "" {
			shouldBeMuted := client.Role != currentTurn
			client.IsMuted = shouldBeMuted
			
			log.Printf("Client %s (role: %s) should be muted: %v", client.Username, client.Role, shouldBeMuted)
			
			// Send mute status to each client
			if err := clientConn.WriteJSON(map[string]interface{}{
				"type": "autoMuteStatus",
				"userId": client.UserID,
				"username": client.Username,
				"isMuted": shouldBeMuted,
				"currentTurn": currentTurn,
				"phase": message.Phase,
			}); err != nil {
				log.Printf("WebSocket write error: %v", err)
			}
		}
	}
	room.Mutex.Unlock()

	// Broadcast phase change to other clients
	room.Mutex.Lock()
	for clientConn := range room.Clients {
		if clientConn != conn {
			if err := clientConn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
			}
		}
	}
	room.Mutex.Unlock()
}

// handleTopicChange handles topic changes
func handleTopicChange(room *Room, conn *websocket.Conn, message Message) {
	log.Printf("Broadcasting topic change: %s to other clients", message.Topic)
	// Broadcast topic change to other clients
	room.Mutex.Lock()
	for clientConn := range room.Clients {
		if clientConn != conn {
			if err := clientConn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
			} else {
				log.Printf("Successfully broadcasted topic change to a client")
			}
		}
	}
	room.Mutex.Unlock()
}

// handleRoleSelection handles role selection
func handleRoleSelection(room *Room, conn *websocket.Conn, message Message) {
	log.Printf("Broadcasting role selection: %s to other clients", message.Role)
	
	// Store the role in the client
	room.Mutex.Lock()
	if client, exists := room.Clients[conn]; exists {
		client.Role = message.Role
	}
	room.Mutex.Unlock()
	
	// Broadcast role selection to other clients
	room.Mutex.Lock()
	for clientConn := range room.Clients {
		if clientConn != conn {
			if err := clientConn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
			} else {
				log.Printf("Successfully broadcasted role selection to a client")
			}
		}
	}
	room.Mutex.Unlock()
}

// handleReadyStatus handles ready status
func handleReadyStatus(room *Room, conn *websocket.Conn, message Message) {
	// Broadcast ready status to other clients
	room.Mutex.Lock()
	for clientConn := range room.Clients {
		if clientConn != conn {
			if err := clientConn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
			}
		}
	}
	room.Mutex.Unlock()
}

// handleMuteRequest handles mute requests
func handleMuteRequest(room *Room, conn *websocket.Conn, message Message, client *Client) {
	room.Mutex.Lock()
	client.IsMuted = true
	room.Mutex.Unlock()

	// Broadcast mute status to other clients
	room.Mutex.Lock()
	for clientConn := range room.Clients {
		if clientConn != conn {
			if err := clientConn.WriteJSON(map[string]interface{}{
				"type": "muteStatus",
				"userId": client.UserID,
				"username": client.Username,
				"isMuted": true,
			}); err != nil {
				log.Printf("WebSocket write error: %v", err)
			}
		}
	}
	room.Mutex.Unlock()
}

// handleUnmuteRequest handles unmute requests
func handleUnmuteRequest(room *Room, conn *websocket.Conn, message Message, client *Client) {
	room.Mutex.Lock()
	client.IsMuted = false
	room.Mutex.Unlock()

	// Broadcast unmute status to other clients
	room.Mutex.Lock()
	for clientConn := range room.Clients {
		if clientConn != conn {
			if err := clientConn.WriteJSON(map[string]interface{}{
				"type": "muteStatus",
				"userId": client.UserID,
				"username": client.Username,
				"isMuted": false,
			}); err != nil {
				log.Printf("WebSocket write error: %v", err)
			}
		}
	}
	room.Mutex.Unlock()
}

// getUserDetails fetches user details from database
func getUserDetails(email string) (string, string, error) {
	// Query user document using email
	userCollection := db.MongoClient.Database("DebateAI").Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user struct {
		ID          string `bson:"_id"`
		Email       string `bson:"email"`
		DisplayName string `bson:"displayName"`
	}

	err := userCollection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return "", "", err
	}

	return user.ID, user.DisplayName, nil
}