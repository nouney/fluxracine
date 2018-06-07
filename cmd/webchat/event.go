package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
	"github.com/nouney/fluxracine/pkg/chat"
	"github.com/nouney/fluxracine/pkg/event"
	"github.com/pkg/errors"
)

const (
	actionSendMessage    = "send_message"
	actionReceiveMessage = "receive_message"
)

type websocketEventSource struct {
	conn *websocket.Conn
}

// beware: websocket.Conn supports max 1 reading goroutine and 1 writing goroutine.
// the reading one is below and used by the event dispatcher.
func (ws websocketEventSource) Next() (*event.Event, error) {
	l := log.New(os.Stdout, "websocket-event-source", 0)
	ev := event.Event{}
	_, msg, err := ws.conn.ReadMessage()
	if err != nil {
		ev.Type = event.EventUserLogout
		return &ev, io.EOF
	}
	l.Printf("receive from websocket: %s", string(msg))

	action, err := jsonparser.GetString(msg, "action")
	if err != nil {
		return nil, errors.Wrap(err, "get action")
	}

	ev.Data, _, _, err = jsonparser.Get(msg, "data")
	if err != nil {
		return nil, errors.Wrap(err, "get data")
	}

	switch action {
	case actionSendMessage:
		ev.Type = event.EventUserSendMessage
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
	return &ev, nil
}

type chatSessionEventSource struct {
	sess *chat.Session
}

func (cses chatSessionEventSource) Next() (*event.Event, error) {
	m, err := cses.sess.ReceiveMessage()
	if err != nil {
		return nil, err
	}

	return &event.Event{
		Type: event.EventUserReceiveMessage,
		Data: m,
	}, nil
}
