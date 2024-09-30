package testingx

import (
	"context"

	"go.infratographer.com/permissions-api/pkg/permissions"
	"go.infratographer.com/x/events"
	"go.infratographer.com/x/gidx"
)

// TestPublisherMethod is a type for the methods of the TestPublisher
type TestPublisherMethod string

const (
	// TestPublisherMethodCreate is the method name for CreateAuthRelationships
	TestPublisherMethodCreate TestPublisherMethod = "CreateAuthRelationships"
	// TestPublisherMethodDelete is the method name for DeleteAuthRelationships
	TestPublisherMethodDelete TestPublisherMethod = "DeleteAuthRelationships"
)

// TestPublisherCalledWith records the arguments passed to the TestPublisher
type TestPublisherCalledWith struct {
	Method     TestPublisherMethod
	Topic      string
	ResourceID gidx.PrefixedID
	Relations  []events.AuthRelationshipRelation
}

// TestPublisher is a test publisher implements the permissions.AuthRelationshipRequestHandler
type TestPublisher struct {
	calledWiths []TestPublisherCalledWith
	err         error
}

// TestPublisher implements permissions.AuthRelationshipRequestHandler
var _ permissions.AuthRelationshipRequestHandler = (*TestPublisher)(nil)

// CalledWith returns the arguments passed to the TestPublisher
func (tp *TestPublisher) CalledWith() []TestPublisherCalledWith {
	return tp.calledWiths
}

// CreateAuthRelationships records the arguments passed to the TestPublisher
func (tp *TestPublisher) CreateAuthRelationships(_ context.Context, topic string, resourceID gidx.PrefixedID, relations ...events.AuthRelationshipRelation) error {
	tp.calledWiths = append(tp.calledWiths, TestPublisherCalledWith{
		Method:     TestPublisherMethodCreate,
		Topic:      topic,
		ResourceID: resourceID,
		Relations:  relations,
	})

	return tp.err
}

// DeleteAuthRelationships records the arguments passed to the TestPublisher
func (tp *TestPublisher) DeleteAuthRelationships(_ context.Context, topic string, resourceID gidx.PrefixedID, relations ...events.AuthRelationshipRelation) error {
	tp.calledWiths = append(tp.calledWiths, TestPublisherCalledWith{
		Method:     TestPublisherMethodDelete,
		Topic:      topic,
		ResourceID: resourceID,
		Relations:  relations,
	})

	return tp.err
}

// ContextWithPublisher injects the TestPublisher into the context
func (tp *TestPublisher) ContextWithPublisher(ctx context.Context) context.Context {
	return context.WithValue(ctx, permissions.AuthRelationshipRequestHandlerCtxKey, tp)
}

// GetPublisherFromContext returns the TestPublisher from the context
func GetPublisherFromContext(ctx context.Context) (p *TestPublisher, ok bool) {
	p, ok = ctx.Value(permissions.AuthRelationshipRequestHandlerCtxKey).(*TestPublisher)
	return
}

// NewTestPublisherOption is a type for the options of the NewTestPublisher
type NewTestPublisherOption func(*TestPublisher)

// TestPublisherWithError sets the error for the TestPublisher
func TestPublisherWithError(err error) NewTestPublisherOption {
	return func(tp *TestPublisher) {
		tp.err = err
	}
}

// NewTestPublisher returns a new TestPublisher
func NewTestPublisher(opts ...NewTestPublisherOption) *TestPublisher {
	p := &TestPublisher{}

	for _, opt := range opts {
		opt(p)
	}

	return p
}
