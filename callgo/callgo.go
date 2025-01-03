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

type OnInitBroadcast struct {
	InitID string `json:"InitID"`
	InitName string `json:"InitName"`
}

type OnDisconnect struct {
	DisconnectMemberID string `json:"disconnectID"`
}

type VideoDataReceive struct {
	VideoData string `json:"video"`
}

type VideoDataSend struct {
	DisplayName string `json:"name"`
	MemberID string `json:"memberID"`
	VideoData string `json:"video"`
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
		mySession.broadcast(OnDisconnect{DisconnectMemberID: myMember.MemberID})
		return nil
	})

	// on client connect
		// notify yourself about your ID
	myMember.safeWrite(OnInitSelf{MyID: myMember.MemberID})
		// notify everyone about all the members in the meeting
	for _, member := range mySession.Members {
		mySession.broadcast(OnInitBroadcast{InitID: member.MemberID, InitName: member.DisplayName})
	}
	
	for {
		// RECEIVE
		var video VideoDataReceive
		err := myMember.Connection.ReadJSON(&video)
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}

		// SEND
		mySession.broadcast(VideoDataSend{DisplayName: myDisplayName, MemberID: myMember.MemberID, VideoData: video.VideoData})
	}

	sessions.getSession(sessionID).disconnectMember(myMember, false, "nil")
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