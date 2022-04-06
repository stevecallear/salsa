package bolt

import (
	"context"
	"encoding/binary"
	"errors"

	"go.etcd.io/bbolt"

	"github.com/stevecallear/salsa"
)

type (
	db struct {
		bdb *bbolt.DB
	}

	tx struct {
		bucket *bbolt.Bucket
	}

	item struct {
		key  []byte
		data []byte
	}

	itemType uint8
)

const (
	itemTypeEvent itemType = iota + 1
	itemTypeSnapshot
)

// New returns a new event store backed by boltdb
func New[T any](bdb *bbolt.DB, optFns ...func(*salsa.Options[T])) *salsa.Store[string, T] {
	return salsa.NewStore[string](&db{bdb: bdb}, optFns...)
}

// Read reads most recent state and events for the specified id
func (d *db) Read(ctx context.Context, id string, _ int) (salsa.EncodedState, []salsa.EncodedEvent, error) {
	var state salsa.EncodedState
	var events []salsa.EncodedEvent

	err := d.bdb.View(func(btx *bbolt.Tx) error {
		bu := btx.Bucket([]byte(id))
		if bu == nil {
			return errors.New("not found")
		}

		c := bu.Cursor()

	loop:
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			ver, ityp, etyp := decodeKey(k)
			switch ityp {
			case itemTypeSnapshot:
				state = salsa.EncodedState{
					Version: ver,
					Data:    v,
				}
				break loop
			case itemTypeEvent:
				events = append(events, salsa.EncodedEvent{
					Type:    etyp,
					Version: ver,
					Data:    v,
				})
			default:
				return errors.New("invalid item type")
			}
		}
		return nil
	})
	if err != nil {
		return salsa.EncodedState{}, nil, err
	}

	reverse(events)
	return state, events, nil
}

// Write executes the specified write function within a transaction
func (d *db) Write(ctx context.Context, id string, fn func(salsa.DBTx) error) error {
	return d.bdb.Update(func(btx *bbolt.Tx) error {
		bu, err := btx.CreateBucketIfNotExists([]byte(id))
		if err != nil {
			return err
		}

		tx := &tx{bucket: bu}
		return fn(tx)
	})
}

// Event writes the specified event
func (t *tx) Event(e salsa.EncodedEvent) error {
	k := encodeKey(e.Version, itemTypeEvent, e.Type)
	if b := t.bucket.Get(k); b != nil {
		return errors.New("version conflict")
	}

	return t.bucket.Put(k, e.Data)
}

// State writes the specified state
func (t *tx) State(s salsa.EncodedState) error {
	k := encodeKey(s.Version, itemTypeSnapshot, "")
	if b := t.bucket.Get(k); b != nil {
		return errors.New("version conflict")
	}

	return t.bucket.Put(k, s.Data)
}

func encodeKey(v uint64, t itemType, st string) []byte {
	stb := []byte(st)
	b := make([]byte, len(stb)+9)

	binary.BigEndian.PutUint64(b, v)
	b[8] = byte(t)
	copy(b[9:], stb)

	return b
}

func decodeKey(b []byte) (uint64, itemType, string) {
	v := binary.BigEndian.Uint64(b[:8])
	t := itemType(b[8])
	st := string(b[9:])

	return v, t, st
}

func reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
