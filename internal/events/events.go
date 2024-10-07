package events

import (
	"go.uber.org/zap"
)

// Events represents a collection of relationships.
type Events struct {
	logger *zap.Logger
}

// Events implements the Service interface.
var _ Service = (*Events)(nil)

// Opt represents an option for configuring Relationships.
type Opt func(*Events)

// NewEvents creates a new Relationships instance with the given NATS URL and options.
func NewEvents(opts ...Opt) *Events {
	r := &Events{}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// WithLogger is an option to set the logger for Relationships.
func WithLogger(logger *zap.Logger) Opt {
	return func(e *Events) {
		e.logger = logger
	}
}
