package salsa

type (
	// Aggregate represents an aggregate root
	Aggregate[T any] struct {
		initState VersionedState[T]
		currState T
		events    []Event[T]
	}

	// VersionedState represents versioned aggregate state
	VersionedState[T any] struct {
		Version uint64 `json:"version"`
		State   T      `json:"state"`
	}
)

// NewAggregate returns a new aggregate with the specified initial state
func NewAggregate[T any](s VersionedState[T]) *Aggregate[T] {
	return &Aggregate[T]{
		initState: s,
		currState: s.State,
	}
}

// InitState returns the initial aggregate state
func (a *Aggregate[T]) InitState() VersionedState[T] {
	return a.initState
}

// State returns the current aggregate state
func (a *Aggregate[T]) State() T {
	return a.currState
}

// Events returns all events applied to the initial state
func (a *Aggregate[T]) Events() []Event[T] {
	return a.events
}

// Apply applies the specified event
func (a *Aggregate[T]) Apply(e Event[T]) error {
	ns, err := e.Apply(a.currState)
	if err != nil {
		return err
	}
	a.currState = ns

	if a.events == nil {
		a.events = []Event[T]{e}
	} else {
		a.events = append(a.events, e)
	}

	return nil
}
