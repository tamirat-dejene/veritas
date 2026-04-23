package messaging

import (
	"context"
	"sync"
)

// Router dispatches messages to handlers based on the topic.
// It implements the Handler interface.
type Router struct {
	handlers map[string]Handler
	mu       sync.RWMutex
	middlewares []Middleware
}

// Middleware is a function that wraps a Handler.
type Middleware func(next Handler) Handler

// NewRouter creates a new messaging Router.
func NewRouter(mw ...Middleware) *Router {
	return &Router{
		handlers:    make(map[string]Handler),
		middlewares: mw,
	}
}

// Register adds a handler for a specific topic.
func (r *Router) Register(topic string, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Wrap handler with router-level middlewares
	wrapped := h
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		wrapped = r.middlewares[i](wrapped)
	}

	r.handlers[topic] = wrapped
}

// Handle dispatches the message to the registered topic handler.
func (r *Router) Handle(ctx context.Context, msg Message) error {
	r.mu.RLock()
	h, ok := r.handlers[msg.Topic]
	r.mu.RUnlock()

	if !ok {
		// No handler registered for this topic. 
		// We return nil to acknowledge the message (skip it) or log a warning.
		return nil 
	}

	return h(ctx, msg)
}

// Route returns the list of all registered topics.
func (r *Router) Topics() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	topics := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		topics = append(topics, t)
	}
	return topics
}
