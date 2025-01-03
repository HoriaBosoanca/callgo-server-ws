package callgo

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
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
	
	// Initialize member
	sessionID := r.URL.Query().Get("sessionID")
	if sessionID == "" {
		http.Error(w, "Wrong session ID", http.StatusBadRequest)
		return
	}
	myMember := sessions.getSession(sessionID).addMember(connection)
	myMember.safeWrite(MemberID{MemberID: myMember.MemberID})

	for {
		// RECEIVE
		var receivedVideo VideoDataTransfer
		err := myMember.Connection.ReadJSON(&receivedVideo)
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		// SEND
		session := sessions.Sessions[sessionID]
		session.mu.Lock()
		for _, member := range session.Members {
			member.safeWrite(receivedVideo)
		}
		session.mu.Unlock()
	}

	sessions.getSession(sessionID).disconnectMember(myMember, false, "nil")
}

// HTTP
func makeSession(w http.ResponseWriter, r *http.Request) {
	session := sessions.addSession()
	w.WriteHeader(http.StatusCreated)
	res := InitializeResponse{SessionID: session.SessionID, Password: session.Password}
	json.NewEncoder(w).Encode(res)
}
	
func triggerDisconnect(w http.ResponseWriter, r *http.Request) {
	var disconnectData Disconnect
	err := json.NewDecoder(r.Body).Decode(&disconnectData)
	if err != nil {
		http.Error(w, "Error decoding data", http.StatusBadRequest)
		return
	}
	session := sessions.getSession(disconnectData.SessionID)
	member := session.getMember(disconnectData.MemberID)
	session.disconnectMember(member, true, disconnectData.Password)
}

// upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}