package bolt_test

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/stevecallear/salsa"
	"go.etcd.io/bbolt"

	"github.com/stevecallear/salsa/store/bolt"
)

func TestNew(t *testing.T) {
	const fn = "bolt_test.db"

	db, err := bbolt.Open(fn, 0666, nil)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(fn); err != nil {
			t.Fatal(err)
		}
	}()

	er := salsa.EventResolverFunc[state](func(string) (salsa.Event[state], error) {
		return new(event), nil
	})

	sut := bolt.New(db, salsa.WithResolver[state](er), salsa.WithSnapshotRate[state](5))

	id := uuid.NewString()
	t.Run("should return an error if the aggregate does not exist", func(t *testing.T) {
		_, err := sut.Get(context.Background(), id)
		assertErrorExists(t, err, true)
	})

	t.Run("should write the aggregate", func(t *testing.T) {
		a := new(salsa.Aggregate[state])
		for i := 1; i <= 12; i++ {
			_, err := a.Apply(&event{Amount: i * 10})
			assertErrorExists(t, err, false)
		}

		err := sut.Save(context.Background(), id, a)
		assertErrorExists(t, err, false)
	})

	t.Run("should return an error if a conflict occurs", func(t *testing.T) {
		a := new(salsa.Aggregate[state])
		_, err := a.Apply(&event{Amount: 100})
		assertErrorExists(t, err, false)

		err = sut.Save(context.Background(), id, a)
		assertErrorExists(t, err, true)
	})

	t.Run("should read the aggregate (state)", func(t *testing.T) {
		act, err := sut.Get(context.Background(), id)
		assertErrorExists(t, err, false)

		assertAggregateEqual(t, act, aggregate{
			state: state{Balance: 780},
			versions: salsa.Versions{
				State:   12,
				Initial: 12,
				Current: 12,
			},
		})
	})

	t.Run("should write additional events", func(t *testing.T) {
		a, err := sut.Get(context.Background(), id)
		assertErrorExists(t, err, false)

		for i := 1; i <= 2; i++ {
			_, err := a.Apply(&event{Amount: i * 10})
			assertErrorExists(t, err, false)
		}

		err = sut.Save(context.Background(), id, a)
		assertErrorExists(t, err, false)
	})

	t.Run("should read the aggregate (state and events)", func(t *testing.T) {
		act, err := sut.Get(context.Background(), id)
		assertErrorExists(t, err, false)

		assertAggregateEqual(t, act, aggregate{
			state: state{Balance: 810},
			versions: salsa.Versions{
				State:   12,
				Initial: 14,
				Current: 14,
			},
		})
	})
}

type (
	state struct {
		Balance int `json:"balance"`
	}

	event struct {
		Amount int `json:"amount"`
	}

	aggregate struct {
		state    state
		versions salsa.Versions
		events   []salsa.Event[state]
	}
)

func (e *event) Type() string {
	return "event"
}

func (e *event) Apply(s state) (state, error) {
	s.Balance += e.Amount
	return s, nil
}

func assertErrorExists(t *testing.T, act error, exp bool) {
	if act != nil && !exp {
		t.Errorf("got %v, expected nil", act)
	}
	if act == nil && exp {
		t.Error("got nil, expected an error")
	}
}

func assertAggregateEqual(t *testing.T, act *salsa.Aggregate[state], exp aggregate) {
	a := aggregate{
		state:    act.State(),
		versions: act.Versions(),
		events:   act.Events(),
	}

	if !reflect.DeepEqual(a, exp) {
		t.Errorf("got %v, expected %v", act, exp)
	}
}
