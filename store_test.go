package salsa_test

import (
	"context"
	"testing"

	"github.com/stevecallear/salsa"
)

func TestStore(t *testing.T) {
	const id = "id"

	er := salsa.EventResolverFunc[state](func(string) (salsa.Event[state], error) {
		return new(event), nil
	})

	sut := salsa.NewMemoryStore[string](salsa.WithResolver[state](er), salsa.WithSnapshotRate[state](5))

	t.Run("should return an error if the aggregate does not exist", func(t *testing.T) {
		_, err := sut.Get(context.Background(), id)
		assertErrorExists(t, err, true)
	})

	t.Run("should write the aggregate", func(t *testing.T) {
		a := new(salsa.Aggregate[state])
		for i := 1; i <= 12; i++ {
			err := a.Apply(&event{Amount: i * 10})
			assertErrorExists(t, err, false)
		}

		err := sut.Save(context.Background(), id, a)
		assertErrorExists(t, err, false)
	})

	t.Run("should return an error if a conflict occurs", func(t *testing.T) {
		a := new(salsa.Aggregate[state])
		err := a.Apply(&event{Amount: 100})
		assertErrorExists(t, err, false)

		err = sut.Save(context.Background(), id, a)
		assertErrorExists(t, err, true)
	})

	t.Run("should read the aggregate", func(t *testing.T) {
		act, err := sut.Get(context.Background(), id)
		assertErrorExists(t, err, false)

		assertAggregateEqual(t, act, aggregate{
			initState: salsa.VersionedState[state]{
				Version: 14, // 12 events and 2 snapshots
				State:   state{Balance: 780},
			},
			currState: state{Balance: 780},
		})
	})
}
