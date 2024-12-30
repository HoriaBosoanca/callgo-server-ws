package callgo

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/teris-io/shortid"
)

func HandleEndpoint(router *mux.Router) {
	router.HandleFunc("/ws", OptionsHandler).Methods("OPTIONS")
	router.HandleFunc("/ws", handleWebSockets)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Member struct {
	Connection *websocket.Conn
}

type VideoDataTransfer struct {
	DisplayName string `json:"name"`
	VideoData string `json:"video"`
}

type Password struct {
	Password string `json:"password"`
}

var sessions map[string]map[string]Member = make(map[string]map[string]Member)

// /ws?sessionID=abcd
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
	memberID, password := addMember(sessionID, connection)
	if password != "nil" {
		connection.WriteJSON(Password{Password: password})
	}

	for {
		// RECEIVE
		var receivedVideo VideoDataTransfer
		err := connection.ReadJSON(&receivedVideo)
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		// SEND
		for _, member := range sessions[sessionID] {
			err = member.Connection.WriteJSON(receivedVideo)
			if err != nil {
				log.Println("Error writing message:", err)
				break
			}
		}
	}

	disconnect(sessionID, memberID)
}

func addMember(sessionID string, connection *websocket.Conn) (memberID string, password string) {
	session, exists := sessions[sessionID]
	password = "nil"
	if !exists {
		session = make(map[string]Member)
		password = shortid.MustGenerate()
	}
	memberID = shortid.MustGenerate()
	session[memberID] = Member{Connection: connection}
	sessions[sessionID] = session 
	return memberID, password
}

func disconnect(sessionID string, memberID string) {
	session, exists := sessions[sessionID]
	if !exists {
		log.Println("Session doesn't exist:", sessionID, memberID)
		return
	}
	member, exists2 := session[memberID]
	if !exists2 {
		log.Println("Member doesn't exist:", sessionID, memberID)
		return
	}

	member.Connection.Close()
	delete(session, memberID)
	sessions[sessionID] = session

	if len(session) == 0 {
		delete(sessions, sessionID)
	}
}