package callgo

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/teris-io/shortid"
)

type Sessions struct {
	Sessions map[string]*Session
	mu sync.Mutex
}

var sessions Sessions = Sessions{Sessions: make(map[string]*Session)}

func (s *Sessions) addSession() (session *Session) {
	sessionID := shortid.MustGenerate()
	sessions.mu.Lock()
	defer sessions.mu.Unlock()
	session = &Session{Members: make(map[string]*Member), SessionID: sessionID, Password: shortid.MustGenerate()}
	sessions.Sessions[sessionID] = session
	return session
}

func (s *Sessions) getSession(sessionID string) (session *Session) {
	session, exists := s.Sessions[sessionID]
	if !exists {
		log.Println("Session not found:", sessionID)
		return &Session{}
	}
	return session
}

type Session struct {
	Members map[string]*Member
	mu sync.Mutex
	SessionID string
	Password string
}

func (s *Session) addMember(conn *websocket.Conn, displayName string) (member *Member) {
	memberID := shortid.MustGenerate()
	s.mu.Lock()
	member = &Member{Connection: conn, MemberID: memberID, DisplayName: displayName}
	s.Members[memberID] = member
	s.mu.Unlock()
	return member
}

func (s *Session) getMember(memberID string) (member *Member) {
	member, exists := s.Members[memberID]
	if !exists {
		log.Println("Member does not exist:", memberID)
		return &Member{}
	}
	return member
}

func (s *Session) disconnectMember(member *Member, requiresAuth bool, password string) {
	if requiresAuth && !s.auth(password) {
		log.Println("Auth func failed:", password)
		return
	}

	s.mu.Lock()
	member.Connection.Close()
	delete(s.Members, member.MemberID)
	s.mu.Unlock()
	
	// on (intentional) client disconnect
	s.broadcast(OnDisconnect{DisconnectMemberID: member.MemberID})
}

func (s *Session) broadcast(data interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, member := range s.Members {
		member.safeWrite(data)
	}
}

func (s *Session) auth(password string) (succes bool) {
	if s.Password == password {
		return true
	} else {
		log.Println("Wrong password:", password)
		return false
	}
}

type Member struct {
	Connection *websocket.Conn
	mu sync.Mutex
	MemberID string
	DisplayName string
}

func (m *Member) safeWrite(data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	err := m.Connection.WriteJSON(data)
	if err != nil {
		log.Println("Error writing to connection:", err)
	}
}