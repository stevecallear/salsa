package salsa_test

import (
	"testing"

	"github.com/stevecallear/salsa"
)

func TestNewAggregate(t *testing.T) {
	evts := []salsa.Event[state]{
		&event{Amount: 5},
		&event{Amount: 10},
	}

	tests := []struct {
		name   string
		state  salsa.VersionedState[state]
		events []salsa.Event[state]
		exp    aggregate
		err    bool
	}{
		{
			name: "should return errors",
			state: salsa.VersionedState[state]{
				Version: 4,
				State:   state{Balance: 10},
			},
			events: []salsa.Event[state]{new(errEvent)},
			err:    true,
		},
		{
			name: "should return the aggregate",
			state: salsa.VersionedState[state]{
				Version: 4,
				State:   state{Balance: 10},
			},
			events: evts,
			exp: aggregate{
				state: state{Balance: 25},
				versions: salsa.Versions{
					State:   4,
					Initial: 6,
					Current: 6,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			act, err := salsa.NewAggregate(tt.state, tt.events...)
			assertErrorExists(t, err, tt.err)
			if err != nil {
				return
			}

			assertAggregateEqual(t, act, tt.exp)
		})
	}
}

func TestAggregate_Apply(t *testing.T) {
	evts := []salsa.Event[state]{
		&event{Amount: 5},
		&event{Amount: 10},
	}

	tests := []struct {
		name   string
		sut    *salsa.Aggregate[state]
		events []salsa.Event[state]
		exp    aggregate
		err    bool
	}{
		{
			name:   "should return errors",
			sut:    new(salsa.Aggregate[state]),
			events: []salsa.Event[state]{new(errEvent)},
			err:    true,
		},
		{
			name: "should apply the events",
			sut: newAggregate(salsa.VersionedState[state]{
				Version: 4,
				State:   state{Balance: 10},
			}),
			events: evts,
			exp: aggregate{
				state: state{Balance: 25},
				versions: salsa.Versions{
					State:   4,
					Initial: 4,
					Current: 6,
				},
				events: evts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, e := range tt.events {
				v, err := tt.sut.Apply(e)
				assertErrorExists(t, err, tt.err)

				if act, exp := v, tt.sut.Versions().Current; act != exp {
					t.Errorf("got %d, expected %d", act, exp)
				}
			}

			assertAggregateEqual(t, tt.sut, tt.exp)
		})
	}
}

type aggregate struct {
	state    state
	versions salsa.Versions
	events   []salsa.Event[state]
}

func assertAggregateEqual(t *testing.T, act *salsa.Aggregate[state], exp aggregate) {
	assertDeepEqual(t, aggregate{
		state:    act.State(),
		versions: act.Versions(),
		events:   act.Events(),
	}, exp)
}

func newAggregate(s salsa.VersionedState[state], es ...salsa.Event[state]) *salsa.Aggregate[state] {
	a, err := salsa.NewAggregate(s, es...)
	if err != nil {
		panic(err)
	}
	return a
}
