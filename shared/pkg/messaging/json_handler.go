package messaging

import (
	"context"
	"encoding/json"
	"fmt"
)

// RegisterJSONHandler registers a handler that automatically unmarshals JSON payloads
// into the specified type T before calling the domain handler fn.
func RegisterJSONHandler[T any](router *Router, topic string, fn func(context.Context, T) error) {
	router.Register(topic, func(ctx context.Context, msg Message) error {
		var payload T
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			return fmt.Errorf("messaging: unmarshal %s: %w", topic, err)
		}
		return fn(ctx, payload)
	})
}
