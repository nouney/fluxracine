// Package event provides a lightweight event system for the chat.
package event

import (
	"io"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Type is a type to distinguish events
type Type int

const (
	// EventUnknown is the default event type
	EventUnknown Type = iota
	// EventUserLogin is triggered when an user logs in
	EventUserLogin
	// EventUserLogout is triggered when an user is disconnected from the server
	EventUserLogout
	// EventUserSendMessage is triggered when an user sends a message to another
	EventUserSendMessage
	// EventUserReceiveMessage is triggered when an user receives a message from another
	EventUserReceiveMessage
)

// Handler is a callback that responses to an event
type Handler = func(interface{}) error

// Dispatcher listens to one or several event sources and executes
// handlers.
type Dispatcher struct {
	sync.Mutex

	handlers map[Type]Handler
	sources  []Source
}

// NewDispatcher creates a new Dispatcher object.
func NewDispatcher(srcs ...Source) *Dispatcher {
	d := Dispatcher{
		handlers: make(map[Type]Handler),
		sources:  srcs,
	}
	return &d
}

// Handle assigns an Handler to an Type.
// Thread-safe
func (d *Dispatcher) Handle(et Type, cb Handler) {
	d.Lock()
	d.handlers[et] = cb
	d.Unlock()
}

// Event is composed of a type and some data.
// The event handler should know the underlying data type.
type Event struct {
	Type Type
	Data interface{}
}

// Listen listens to all events sources and executes matching handlers.
func (d *Dispatcher) Listen() error {
	out := make(chan *Event)
	var wg sync.WaitGroup
	wg.Add(len(d.sources))

	for _, src := range d.sources {
		go func(src Source) {
			err := d.listenSource(src, out)
			if err != io.EOF {
				log.Error(errors.Wrap(err, "listen source:"))
			}
			wg.Done()
		}(src)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	for {
		event, ok := <-out
		if !ok {
			break
		}

		d.Lock()
		cb, ok := d.handlers[event.Type]
		d.Unlock()
		if !ok {
			continue
		}

		err := cb(event.Data)
		if err != nil {
			log.Println(err)
		}
	}
	return nil
}

// listenSource listens to a specific source.
func (d *Dispatcher) listenSource(src Source, out chan<- *Event) error {
	for {
		event, err := src.Next()
		if event != nil {
			out <- event
		}
		if err != nil {
			return err
		}
	}
}

// Source is an interface that represents a source of event.
// It is used by the dispatcher, which just calls Next in a loop.
// Implementation of Next should be blocking until the next event occurs.
type Source interface {
	// Wait until an event occurs.
	// Must return io.EOF when finished.
	Next() (*Event, error)
}
