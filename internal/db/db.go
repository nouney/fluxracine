package db

import "errors"

// DB is the database used by the chat servers
type DB interface {
	// AssignServer assigns a server to a user
	AssignServer(nickname, server string) error
	// GetServer retrieves the server associated to the user
	GetServer(nickname string) (string, error)
	// UnassignServer un-assigns a server from a user
	UnassignServer(nickname string) error
}

var (
	ErrNotFound = errors.New("not found")
)
