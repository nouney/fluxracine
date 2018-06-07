package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/dustinkirkland/golang-petname"

	"github.com/nouney/fluxracine/internal/db"
	"github.com/pkg/errors"
)

var (
	// ErrUserNotFound is returned when the user is not found.
	ErrUserNotFound = errors.New("not found")
)

// Server is a server that provides a one-to-one chat system.
type Server struct {
	// db used by the server
	db db.DB
	// address of form "ip:port" of the internal http server
	httpAddr string
	sessions map[string]*Session
	mutex    *sync.Mutex
	// Fake session of user "SYSTEM"
	systemSess *Session
}

// NewServer creates a new Server object.
func NewServer(db db.DB, opts ...Opt) (*Server, error) {
	s := &Server{
		db:       db,
		sessions: make(map[string]*Session),
		httpAddr: "localhost:8000",
		mutex:    new(sync.Mutex),
	}

	for _, opt := range opts {
		err := opt(s)
		if err != nil {
			return nil, err
		}
	}

	// session used by the server to send messages as "SYSTEM"
	s.systemSess = &Session{
		server:   s,
		Nickname: "SYSTEM",
	}
	return s, nil
}

// NewSession creates a new session bound to this Server
func (s *Server) NewSession() (*Session, error) {
	sess := &Session{
		server: s,
		recv:   make(chan *MessagePayload, 10),
	}

	// generate a random nickname and assign the server address to the user
	nickname := petname.Generate(2, "-")
	err := s.db.AssignServer(nickname, s.httpAddr)
	if err != nil {
		// check if dup
		// if dup: retry with another name
		return nil, err
	}
	sess.Nickname = nickname
	s.mutex.Lock()
	s.sessions[nickname] = sess
	s.mutex.Unlock()

	// greets the user and send its nickname
	err = s.systemSess.SendMessage(nickname, fmt.Sprintf("Greetings, %s.", nickname))
	if err != nil {
		return nil, errors.Wrap(err, "send to user")
	}
	return sess, nil
}

// CloseSession closes a session.
// It removes the user from the entire system.
func (s *Server) CloseSession(nickname string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := s.db.UnassignServer(nickname)
	if err != nil {
		return errors.Wrap(err, "db")
	}

	sess := s.sessions[nickname]
	if sess != nil {
		close(sess.recv)
	}
	delete(s.sessions, nickname)
	return nil
}

// MessagePayload represents a message
type MessagePayload struct {
	From    string
	To      string
	Message string
}

// Send sends a message from a user to another one.
// If the receiver is not connected on this server, the message will be forwarded
// to the appropriate server.
func (s Server) Send(m *MessagePayload) error {
	err := s.sendToUser(m)
	if err == nil {
		return nil
	}
	if err != ErrUserNotFound {
		return errors.Wrap(err, "send to user")
	}

	err = s.forwardMessage(m)
	if err != nil {
		if err == ErrUserNotFound {
			return err
		}
		return errors.Wrap(err, "forward")
	}
	return nil
}

// Receive waits until a message is received by user nickname.
// Thread safe.
func (s Server) Receive(nickname string) (*MessagePayload, error) {
	s.mutex.Lock()
	sess, ok := s.sessions[nickname]
	s.mutex.Unlock()
	if !ok {
		return nil, ErrUserNotFound
	}
	m, ok := <-sess.recv
	if !ok {
		return nil, io.EOF
	}
	return m, nil
}

// Run runs the server.
// At this time, it just runs the internal HTTP server in background.
func (s Server) Run() {
	http.HandleFunc("/send", s.sendHandler)
	go http.ListenAndServe(s.httpAddr, nil)
}

// NbSessions returns the number of session on the server.
func (s Server) NbSessions() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return len(s.sessions)
}

// sendToUser send a message to a user connected on this server, using its channel.
// Returns ErrUserNotFound if the user is not connected on this server.
func (s Server) sendToUser(m *MessagePayload) error {
	s.mutex.Lock()
	sess := s.sessions[m.To]
	s.mutex.Unlock()
	if sess == nil {
		log.Printf("user \"%s\" is not connected on this server", m.To)
		return ErrUserNotFound
	}

	sess.recv <- m
	return nil
}

// sendHandler is the HTTP handler used when another server needs this server to send a message
// to a user (forwarding).
func (s Server) sendHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("read all:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	m := MessagePayload{}
	err = json.Unmarshal(body, &m)
	if err != nil {
		log.Println("unmarshal json:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Printf("message to forward: %+v", &m)
	err = s.sendToUser(&m)
	if err == ErrUserNotFound {
		w.WriteHeader(http.StatusNotFound)
	}
}

// forwardMessage forwards a message to the appropriate server so it can be sent to the user.
// Returns ErrUserNotFound if no server is associated to the receiver or if the user doesn't
// exist on the server.
func (s Server) forwardMessage(m *MessagePayload) error {
	// retrieve the server on which the receiver is connected
	server, err := s.db.GetServer(m.To)
	if err != nil {
		if err == db.ErrNotFound {
			s.systemSess.SendMessage(m.From, fmt.Sprintf("user \"%s\": not found", m.To))
			return ErrUserNotFound
		}
		return err
	}

	log.Printf("forward message %+v to \"%s\"", m, server)

	b, err := json.Marshal(m)
	if err != nil {
		return errors.Wrap(err, "json marshal")
	}

	resp, err := http.Post("http://"+server+"/send", "application/json", bytes.NewBuffer(b))
	if err != nil {
		return errors.Wrap(err, "http post")
	}
	if resp.StatusCode == http.StatusNotFound {
		s.systemSess.SendMessage(m.From, fmt.Sprintf("user \"%s\": not found", m.To))
		return ErrUserNotFound
	}
	if resp.StatusCode == http.StatusInternalServerError {
		return fmt.Errorf("http post bad status code: %d", resp.StatusCode)
	}
	return nil
}
