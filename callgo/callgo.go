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

// type VideoDataReceive struct {
// 	VideoData string `json:"video"`
// }

// type VideoDataSend struct {
// 	DisplayName string `json:"name"`
// 	MemberID string `json:"memberID"`
// 	VideoData string `json:"video"`
// }

type SDPmessage struct {
	To string `json:"to"`
	From string `json:"from"`
	SDP SDP `json:"sdp"`
}

type SDP struct {
	Type string `json:"type"`
	SDP string `json:"sdp"`
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
		// RECEIVE
		var sdp SDPmessage
		err := myMember.Connection.ReadJSON(&sdp)
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		// SEND
		mySession.getMember(sdp.To).safeWrite(sdp)
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