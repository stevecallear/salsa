package salsa

type (
	// Aggregate represents an aggregate root
	Aggregate[T any] struct {
		state    T
		versions Versions
		events   []Event[T]
	}

	// Versions represents aggergate versions
	Versions struct {
		State, Initial, Current uint64
	}

	// VersionedState represents versioned aggregate state
	VersionedState[T any] struct {
		Version uint64 `json:"version"`
		State   T      `json:"state"`
	}
)

// NewAggregate returns a new aggregate with the specified initial state
func NewAggregate[T any](s VersionedState[T], es ...Event[T]) (*Aggregate[T], error) {
	a := &Aggregate[T]{
		state: s.State,
		versions: Versions{
			State:   s.Version,
			Initial: s.Version,
			Current: s.Version,
		},
	}

	var err error
	for _, e := range es {
		a.state, err = e.Apply(a.state)
		if err != nil {
			return nil, err
		}

		a.versions.Initial++
		a.versions.Current++
	}

	return a, nil
}

// State returns the current aggregate state
func (a *Aggregate[T]) State() T {
	return a.state
}

// Versions returns the aggregate versions
func (a *Aggregate[T]) Versions() Versions {
	return a.versions
}

// Events returns all events applied to the initial state
func (a *Aggregate[T]) Events() []Event[T] {
	return a.events
}

// Apply applies the specified event
func (a *Aggregate[T]) Apply(e Event[T]) (uint64, error) {
	ns, err := e.Apply(a.state)
	if err != nil {
		return 0, err
	}

	a.state = ns
	a.versions.Current++

	if a.events == nil {
		a.events = []Event[T]{e}
	} else {
		a.events = append(a.events, e)
	}

	return a.versions.Current, nil
}
