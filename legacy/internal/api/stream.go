package api

import (
	"log"
	
	"github.com/openclaw/sentinel-backend/internal/model"
)

// StreamBroker manages Server-Sent Events connections
type StreamBroker struct {
	clients    map[chan *model.Event]bool
	register   chan chan *model.Event
	unregister chan chan *model.Event
	broadcast  chan *model.Event
}

// NewStreamBroker creates a new stream broker
func NewStreamBroker() *StreamBroker {
	broker := &StreamBroker{
		clients:    make(map[chan *model.Event]bool),
		register:   make(chan chan *model.Event),
		unregister: make(chan chan *model.Event),
		broadcast:  make(chan *model.Event, 100),
	}
	go broker.run()
	return broker
}

// run handles client registration, unregistration, and broadcasting
func (b *StreamBroker) run() {
	for {
		select {
		case client := <-b.register:
			b.clients[client] = true
			log.Printf("StreamBroker: Client registered. Total clients: %d", len(b.clients))
		case client := <-b.unregister:
			if _, ok := b.clients[client]; ok {
				delete(b.clients, client)
				close(client)
				log.Printf("StreamBroker: Client unregistered. Total clients: %d", len(b.clients))
			}
		case event := <-b.broadcast:
			log.Printf("StreamBroker: Broadcasting event %s to %d clients", event.ID, len(b.clients))
			for client := range b.clients {
				select {
				case client <- event:
					// event sent successfully
					log.Printf("StreamBroker: Event %s sent to client", event.ID)
				default:
					// client buffer full, skip to avoid blocking
					log.Printf("StreamBroker: Client buffer full, skipping event %s", event.ID)
				}
			}
		}
	}
}

// NewClient registers a new SSE client and returns a channel to receive events
func (b *StreamBroker) NewClient() chan *model.Event {
	client := make(chan *model.Event, 10)
	b.register <- client
	return client
}

// RemoveClient unregisters a client
func (b *StreamBroker) RemoveClient(client chan *model.Event) {
	b.unregister <- client
}

// Broadcast sends an event to all connected clients
func (b *StreamBroker) Broadcast(event *model.Event) {
	log.Printf("StreamBroker.Broadcast: Event %s from %s, sending to broadcast channel", event.ID, event.Source)
	b.broadcast <- event
}