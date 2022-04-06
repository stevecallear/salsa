package salsa

import (
	"context"
	"errors"
)

type (
	// Store represents an event store
	Store[TI comparable, TS any] struct {
		opts Options[TS]
		db   DB[TI]
	}

	// Options represents a set of store options
	Options[TS any] struct {
		SnapshotRate  int
		Encoder       Encoder
		Decoder       Decoder
		EventResolver EventResolver[TS]
	}

	// EncodedState represents encoded state
	EncodedState struct {
		Version uint64
		Data    []byte
	}

	// Encoded event represents an encoded event
	EncodedEvent struct {
		Type    string
		Version uint64
		Data    []byte
	}

	// DB represents an events DB
	DB[TI comparable] interface {
		Read(ctx context.Context, id TI, limit int) (EncodedState, []EncodedEvent, error)
		Write(ctx context.Context, id TI, fn func(DBTx) error) error
	}

	// DBTx represents an events DB transaction
	DBTx interface {
		Event(e EncodedEvent) error
		State(s EncodedState) error
	}
)

// NewStore returns a new event store backed by the specified DB
func NewStore[TI comparable, TS any](db DB[TI], optFns ...func(*Options[TS])) *Store[TI, TS] {
	o := Options[TS]{
		SnapshotRate: 10,
		Encoder:      EncodeJSON,
		Decoder:      DecodeJSON,
		EventResolver: EventResolverFunc[TS](func(string) (Event[TS], error) {
			return nil, errors.New("invalid event type")
		}),
	}

	for _, fn := range optFns {
		fn(&o)
	}

	return &Store[TI, TS]{
		opts: o,
		db:   db,
	}
}

// Get retrieves the aggregate with the specified id
func (s *Store[TI, TS]) Get(ctx context.Context, id TI) (*Aggregate[TS], error) {
	es, ees, err := s.db.Read(ctx, id, s.opts.SnapshotRate)
	if err != nil {
		return nil, err
	}

	if es.Data == nil && len(ees) < 1 {
		return nil, errors.New("not found")
	}

	var vs VersionedState[TS]
	if es.Data != nil {
		if err = s.opts.Decoder.Decode(es.Data, &vs.State); err != nil {
			return nil, err
		}
		vs.Version = es.Version
	}

	for _, ee := range ees {
		e, err := s.opts.EventResolver.Resolve(ee.Type)
		if err != nil {
			return nil, err
		}

		if err = s.opts.Decoder.Decode(ee.Data, e); err != nil {
			return nil, err
		}

		vs.State, err = e.Apply(vs.State)
		if err != nil {
			return nil, err
		}

		vs.Version++
	}

	return &Aggregate[TS]{
		initState: vs,
		currState: vs.State,
	}, nil
}

// Save saves the specified aggregate
func (s *Store[TI, TS]) Save(ctx context.Context, id TI, a *Aggregate[TS]) error {
	vs := a.InitState()

	var err error
	var b []byte
	return s.db.Write(ctx, id, func(tx DBTx) error {
		for _, e := range a.Events() {
			vs.State, err = e.Apply(vs.State)
			if err != nil {
				return err
			}
			vs.Version++

			b, err = s.opts.Encoder.Encode(e)
			if err != nil {
				return err
			}

			if err = tx.Event(EncodedEvent{
				Type:    e.Type(),
				Version: vs.Version,
				Data:    b,
			}); err != nil {
				return err
			}

			if vs.Version%uint64(s.opts.SnapshotRate) == 0 {
				vs.Version++
				b, err = s.opts.Encoder.Encode(vs.State)
				if err != nil {
					return err
				}

				if err = tx.State(EncodedState{
					Version: vs.Version,
					Data:    b,
				}); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// WithSnapshotRate configures the store to snapshot at the specified rate
func WithSnapshotRate[T any](rate int) func(*Options[T]) {
	return func(o *Options[T]) {
		o.SnapshotRate = rate
	}
}
