package salsa

type (
	// Event represents an event
	Event[T any] interface {
		Type() string
		Apply(state T) (T, error)
	}

	// EventResolver represents an event resolver
	EventResolver[T any] interface {
		Resolve(eventType string) (Event[T], error)
	}

	// EventResolverFunc represents an event resolver func
	EventResolverFunc[T any] func(eventType string) (Event[T], error)
)

// Resolve resolves the event for the specified type
func (r EventResolverFunc[T]) Resolve(eventType string) (Event[T], error) {
	return r(eventType)
}

// WithResolver configures the store to use the specified event resolver
func WithResolver[T any](r EventResolver[T]) func(*Options[T]) {
	return func(o *Options[T]) {
		o.EventResolver = r
	}
}
