package salsa

import (
	"context"
	"errors"
	"sync"
)

type (
	memDB[T comparable] struct {
		items map[T][]memDBItem
		mu    sync.RWMutex
	}

	memTX struct {
		version uint64
		items   []memDBItem
		mu      sync.Mutex
	}

	memDBItem struct {
		itype   memDBItemType
		etype   string
		version uint64
		data    []byte
	}

	memDBItemType uint8
)

const (
	memDBItemTypeEvent memDBItemType = iota + 1
	memDBItemTypeState
)

// NewMemoryStore returns a new in-memory event store
func NewMemoryStore[TI comparable, TS any](optFns ...func(*Options[TS])) *Store[TI, TS] {
	return NewStore[TI](new(memDB[TI]), optFns...)
}

// Read returns the initial state and events for the specified aggregate
func (db *memDB[T]) Read(ctx context.Context, id T, _ int) (EncodedState, []EncodedEvent, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	items := db.items[id]
	if len(items) < 1 {
		return EncodedState{}, nil, errors.New("not found")
	}

	var state EncodedState
	var events []EncodedEvent

loop:
	for i := len(items) - 1; i >= 0; i-- {
		switch items[i].itype {
		case memDBItemTypeState:
			state = EncodedState{
				Version: items[i].version,
				Data:    items[i].data,
			}
			break loop
		case memDBItemTypeEvent:
			events = append(events, EncodedEvent{
				Type:    items[i].etype,
				Version: items[i].version,
				Data:    items[i].data,
			})
		default:
			return EncodedState{}, nil, errors.New("invalid item type")
		}
	}

	reverse(events)
	return state, events, nil
}

// Write writes the specified values to the store
func (db *memDB[T]) Write(ctx context.Context, id T, fn func(DBTx) error) error {
	var pv uint64
	if len(db.items[id]) > 0 {
		pv = db.items[id][len(db.items[id])-1].version
	}

	tx := &memTX{version: pv}
	if err := fn(tx); err != nil {
		return err
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if db.items == nil {
		db.items = map[T][]memDBItem{}
	}

	db.items[id] = append(db.items[id], tx.items...)
	return nil
}

func (tx *memTX) Event(e EncodedEvent) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if e.Version != tx.version+1 {
		return errors.New("version conflict")
	}

	tx.version++
	tx.items = append(tx.items, memDBItem{
		itype:   memDBItemTypeEvent,
		etype:   e.Type,
		version: e.Version,
		data:    e.Data,
	})

	return nil
}

func (tx *memTX) State(s EncodedState) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if s.Version != tx.version {
		return errors.New("version conflict")
	}

	tx.version++
	tx.items = append(tx.items, memDBItem{
		itype:   memDBItemTypeState,
		version: s.Version,
		data:    s.Data,
	})

	return nil
}

func reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
