package chat

import "io"

// Session is an user chat session.
// It is created each time a user logs in.
type Session struct {
	Nickname string

	server *Server
	recv   chan *MessagePayload
}

// SendMessage sends a message to someone.
func (s *Session) SendMessage(to, msg string) error {
	return s.server.Send(&MessagePayload{
		From:    s.Nickname,
		To:      to,
		Message: msg,
	})
}

// ReceiveMessage waits until it receives a message.
func (s *Session) ReceiveMessage() (*MessagePayload, error) {
	m, ok := <-s.recv
	if !ok {
		return nil, io.EOF
	}
	return m, nil
}

// Close closes the session.
// The object cannot be reused after.
func (s *Session) Close() error {
	return s.server.CloseSession(s.Nickname)
}
