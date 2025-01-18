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

type OnInitSelf struct {
	MyID string `json:"myID"`
}

type MemberNotification struct {
	Type string `json:"type"`
	MemberID string `json:"memberID"`
	MemberName string `json:"memberName"`
}

type MessageType struct {
	Type string `json:"type"`
}

type SDPmessage struct {
	Type string `json:"type"`
	To string `json:"to"`
	From string `json:"from"`
	SDP SDP `json:"sdp"`
}

type SDP struct {
	Type string `json:"type"`
	SDP string `json:"sdp"`
}

type ICEmessage struct {
	Type string `json:"type"`
	To string `json:"to"`
	From string `json:"from"`
	ICE ICE `json:"ice"`
}

type ICE struct {
	Candidate string `json:"candidate"`
	SdpMid string `json:"sdpMid"`
	SdpMLineIndex int `json:"sdpMLineIndex"` 
	Foundation string `json:"foundation"`
	Component string `json:"component"`
	Priority int `json:"priority"`
	Address string `json:"address"`
	Protocol string `json:"protocol"`
	Port int `json:"port"`
	Type string `json:"type"`
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
	myDisplayName := r.URL.Query().Get("displayName")
	if myDisplayName == "" {
		http.Error(w, "Wrong session ID", http.StatusBadRequest)
		return
	}
	mySession := sessions.getSession(sessionID)
	myMember := mySession.addMember(connection, myDisplayName)
	
	// on (unexpecred) client disconnect, notify everyone else
	myMember.Connection.SetCloseHandler(func(code int, text string) error {
		delete(mySession.Members, myMember.MemberID)
		mySession.broadcast(MemberNotification{Type: "leave", MemberID: myMember.MemberID, MemberName: myMember.DisplayName})
		return nil
	})

	// on client connect
		// notify self about what ID you have
	myMember.safeWrite(MemberNotification{Type: "assignID", MemberID: myMember.MemberID, MemberName: myMember.DisplayName})	
		// notify self about who is already in meeting
	for _, member := range mySession.Members {
		if member.MemberID != myMember.MemberID {
			myMember.safeWrite(MemberNotification{Type: "exist", MemberName: member.DisplayName, MemberID: member.MemberID})
		}
	}
		// notify everyone else about the new member
	for _, member := range mySession.Members {
		if member.MemberID != myMember.MemberID {
			member.safeWrite(MemberNotification{Type: "join", MemberName: myMember.DisplayName, MemberID: myMember.MemberID})
		}
	}

	defer sessions.getSession(sessionID).disconnectMember(myMember, false, "nil")
	
	for {
		_, message, err := myMember.Connection.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		var messageType MessageType
		err = json.Unmarshal(message, &messageType)
		if err != nil {
			log.Println("Error unmarshalling to message type:", err)
			break
		}

		switch messageType.Type {
		case "sdp":
			var sdp SDPmessage
			err = json.Unmarshal(message, &sdp)
			if err != nil {
				log.Println("Error unmarshalling sdp:", err)
				break
			} 
			mySession.getMember(sdp.To).safeWrite(sdp)
		case "ice":
			var ice ICEmessage
			err = json.Unmarshal(message, &ice)
			if err != nil {
				log.Println("Error unmarshalling ice:", err)
			}
			mySession.getMember(ice.To).safeWrite(ice)
		default:
			log.Println("Unknown message type:", messageType.Type)
		}
	}
}

// HTTP
type InitializeResponse struct {
	SessionID string `json:"sessionID"`
	Password string `json:"password"`
}

func makeSession(w http.ResponseWriter, r *http.Request) {
	session := sessions.addSession()
	w.WriteHeader(http.StatusCreated)
	res := InitializeResponse{SessionID: session.SessionID, Password: session.Password}
	json.NewEncoder(w).Encode(res)
}
	
type Disconnect struct {
	SessionID string `json:"sessionID"`
	MemberID string `json:"memberID"`
	Password string `json:"password"`
}

func triggerDisconnect(w http.ResponseWriter, r *http.Request) {
	var disconnectData Disconnect
	err := json.NewDecoder(r.Body).Decode(&disconnectData)
	if err != nil {
		http.Error(w, "Error decoding disconnect data", http.StatusBadRequest)
		return
	}
	session := sessions.getSession(disconnectData.SessionID)
	member := session.getMember(disconnectData.MemberID)
	session.disconnectMember(member, true, disconnectData.Password)
	w.WriteHeader(http.StatusNoContent)
}

// upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}