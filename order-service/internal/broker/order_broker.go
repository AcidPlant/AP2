package broker

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/lib/pq"
)

type StatusEvent struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

type OrderBroker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan StatusEvent
}

func NewOrderBroker() *OrderBroker {
	return &OrderBroker{
		subscribers: make(map[string][]chan StatusEvent),
	}
}

func (b *OrderBroker) Subscribe(orderID string) chan StatusEvent {
	ch := make(chan StatusEvent, 4)
	b.mu.Lock()
	b.subscribers[orderID] = append(b.subscribers[orderID], ch)
	b.mu.Unlock()
	return ch
}

func (b *OrderBroker) Unsubscribe(orderID string, ch chan StatusEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.subscribers[orderID]
	for i, s := range subs {
		if s == ch {
			b.subscribers[orderID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			return
		}
	}
}

func (b *OrderBroker) Publish(event StatusEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscribers[event.OrderID] {
		select {
		case ch <- event:
		default:
		}
	}
}

func (b *OrderBroker) ListenAndForward(dsn string) {
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("pq listener error: %v", err)
		}
	}

	listener := pq.NewListener(dsn, 10*time.Second, time.Minute, reportProblem)
	if err := listener.Listen("order_status_channel"); err != nil {
		log.Fatalf("pg listen: %v", err)
	}
	log.Println("order broker: listening on order_status_channel")

	for n := range listener.Notify {
		if n == nil {
			continue
		}
		var event StatusEvent
		if err := json.Unmarshal([]byte(n.Extra), &event); err != nil {
			log.Printf("broker: bad payload: %v", err)
			continue
		}
		b.Publish(event)
	}
}
