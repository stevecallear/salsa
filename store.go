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
		Read(ctx context.Context, id TI) (EncodedState, []EncodedEvent, error)
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
	es, ees, err := s.db.Read(ctx, id)
	if err != nil {
		return nil, err
	}

	var vs VersionedState[TS]
	if es.Data != nil {
		if err = s.opts.Decoder.Decode(es.Data, &vs.State); err != nil {
			return nil, err
		}
		vs.Version = es.Version
	}

	des := make([]Event[TS], len(ees))
	for i, ee := range ees {
		de, err := s.opts.EventResolver.Resolve(ee.Type)
		if err != nil {
			return nil, err
		}

		if err = s.opts.Decoder.Decode(ee.Data, de); err != nil {
			return nil, err
		}

		des[i] = de
	}

	return NewAggregate(vs, des...)
}

// Save saves the specified aggregate
func (s *Store[TI, TS]) Save(ctx context.Context, id TI, a *Aggregate[TS]) error {
	var err error
	var b []byte
	return s.db.Write(ctx, id, func(tx DBTx) error {
		v := a.Versions()

		for i, e := range a.Events() {
			b, err = s.opts.Encoder.Encode(e)
			if err != nil {
				return err
			}

			if err = tx.Event(EncodedEvent{
				Type:    e.Type(),
				Version: v.Initial + uint64(i+1),
				Data:    b,
			}); err != nil {
				return err
			}
		}

		if v.Current-v.State > uint64(s.opts.SnapshotRate) {
			b, err = s.opts.Encoder.Encode(a.State())
			if err != nil {
				return err
			}

			if err = tx.State(EncodedState{
				Version: v.Current,
				Data:    b,
			}); err != nil {
				return err
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
