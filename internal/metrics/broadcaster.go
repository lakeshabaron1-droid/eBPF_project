package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)


type SSEBroker struct {
	mu         sync.RWMutex
	clients    map[chan []byte]bool


	register   chan chan []byte
	unregister chan chan []byte

	input      chan AggregatedSnapshot
	done       chan struct{}
}


func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients:    make(map[chan []byte]bool),
		register:   make(chan chan []byte),
		unregister: make(chan chan []byte),
		input:      make(chan AggregatedSnapshot, 64),

		done:       make(chan struct{}),
	}
}

func (b *SSEBroker) InputChannel() chan<- AggregatedSnapshot {
	return b.input
}

func (b *SSEBroker) Start() {
	go b.hub()
}

func (b *SSEBroker) Stop() {
	close(b.done)
}

func (b *SSEBroker) hub() {
	for {
		select {
		case <-b.done:
			b.mu.Lock()
			for ch := range b.clients {


				close(ch)
				delete(b.clients, ch)
			}
			b.mu.Unlock()
			return

		case client := <-b.register:
			b.mu.Lock()
			b.clients[client] = true
			b.mu.Unlock()


		case client := <-b.unregister:
			b.mu.Lock()
			if _, ok := b.clients[client]; ok {


				close(client)
				delete(b.clients, client)
			}
			b.mu.Unlock()




		case snap := <-b.input:
			data, err := json.Marshal(snap)
			if err != nil {
				continue
			}
			b.mu.RLock()
			for client := range b.clients {
				select {
				case client <- data:
				default:
					go func(c chan []byte) {
						b.unregister <- c
					}(client)
				}

			}
			b.mu.RUnlock()
		}
	}

}

func (b *SSEBroker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)

	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	w.Header().Set("Access-Control-Allow-Origin", "*")

	client := make(chan []byte, 16)


