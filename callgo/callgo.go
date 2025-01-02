package callgo

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/teris-io/shortid"
)

// ENDPOINTS
func HandleEndpoint(router *mux.Router) {
	router.HandleFunc("/ws", OptionsHandler).Methods("OPTIONS")
	router.HandleFunc("/disconnect", OptionsHandler).Methods("OPTIONS")
	router.HandleFunc("/initialize", OptionsHandler).Methods("OPTIONS")

	router.HandleFunc("/ws", handleWebSockets)
	router.HandleFunc("/disconnect", triggerDisconnect).Methods("POST")
	router.HandleFunc("/initialize", makeSession).Methods("POST")
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// CONNECTIONS
var sessions map[string]*Session = make(map[string]*Session)

type Session struct {
	Members map[string]*Member
	Password string
}

type Member struct {
	Connection *websocket.Conn
	mu sync.Mutex
}

func (member *Member) safeWrite(data interface{}) {
	member.mu.Lock()
	defer member.mu.Unlock()
	err := member.Connection.WriteJSON(data)
	if err != nil {
		log.Println("Error writing to connection:", err)
	}
}

// JSON 
type MemberID struct {
	MemberID string `json:"memberID"`
}

type VideoDataTransfer struct {
	DisplayName string `json:"name"`
	ID string `json:"ID"`
	VideoData string `json:"video"`
}

type InitializeResponse struct {
	SessionID string `json:"sessionID"`
	Password string `json:"password"`
}

type Disconnect struct {
	SessionID string `json:"sessionID"`
	MemberID string `json:"memberID"`
	Password string `json:"password"`
}

// MAIN WS LOOP
func handleWebSockets(w http.ResponseWriter, r *http.Request) {
	connection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer connection.Close()

	// Initialize member
	sessionID := r.URL.Query().Get("sessionID")
	if sessionID == "" {
		http.Error(w, "Wrong session ID", http.StatusBadRequest)
		return
	}
	memberID := addMember(sessionID, connection)
	connection.WriteJSON(MemberID{MemberID: memberID})

	for {
		// RECEIVE
		var receivedVideo VideoDataTransfer
		err := connection.ReadJSON(&receivedVideo)
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		// SEND
		session := sessions[sessionID]
		for _, member := range session.Members {
			member.safeWrite(receivedVideo)
		}
	}

	disconnect(sessionID, memberID, false, "nil")
}

// HTTP
func makeSession(w http.ResponseWriter, r *http.Request) {
	sessionID := shortid.MustGenerate()
	session := &Session{Members: make(map[string]*Member), Password: shortid.MustGenerate()}
	sessions[sessionID] = session
	
	w.WriteHeader(http.StatusCreated)
	res := InitializeResponse{SessionID: sessionID, Password: session.Password}
	json.NewEncoder(w).Encode(res)
}
	
func triggerDisconnect(w http.ResponseWriter, r *http.Request) {
	var disconnectData Disconnect
	err := json.NewDecoder(r.Body).Decode(&disconnectData)
	if err != nil {
		http.Error(w, "Error decoding data", http.StatusBadRequest)
		return
	}
	disconnect(disconnectData.SessionID, disconnectData.MemberID, true, disconnectData.Password)
}

// UTILITIES
func addMember(sessionID string, connection *websocket.Conn) (memberID string) {
	session := sessions[sessionID]
	memberID = shortid.MustGenerate()
	session.Members[memberID] = &Member{Connection: connection}
	sessions[sessionID] = session
	return memberID
}

func disconnect(sessionID string, memberID string, requiresPassword bool, password string) {
	session, exists := sessions[sessionID]
	if !exists {
		log.Println("Session doesn't exist:", sessionID, memberID)
		return
	}
	member, exists2 := session.Members[memberID]
	if !exists2 {
		log.Println("Member doesn't exist:", sessionID, memberID)
		return
	}

	if requiresPassword && !auth(sessionID, memberID, password) {
		log.Println("Auth func failed", sessionID, memberID, password)
		return
	}

	member.Connection.Close()
	delete(session.Members, memberID)
	sessions[sessionID] = session

	// if len(session.Members) == 0 {
	// 	delete(sessions, sessionID)
	// }
}

func auth(sessionID string, memberID string, password string) (succes bool) {
	session, exists := sessions[sessionID]
	if !exists {
		log.Println("Session not found", sessionID, memberID, password)
		return false
	}
	if session.Password == password {
		return true
	} else {
		log.Println("Wrong password", sessionID, memberID, password)
		return false
	}
}