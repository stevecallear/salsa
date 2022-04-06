package salsa_test

import (
	"testing"

	"github.com/stevecallear/salsa"
)

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
			sut: salsa.NewAggregate(salsa.VersionedState[state]{
				Version: 4,
				State:   state{Balance: 10},
			}),
			events: evts,
			exp: aggregate{
				initState: salsa.VersionedState[state]{
					Version: 4,
					State:   state{Balance: 10},
				},
				currState: state{Balance: 25},
				events:    evts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, e := range tt.events {
				err := tt.sut.Apply(e)
				assertErrorExists(t, err, tt.err)
			}

			assertAggregateEqual(t, tt.sut, tt.exp)
		})
	}
}

type aggregate struct {
	initState salsa.VersionedState[state]
	currState state
	events    []salsa.Event[state]
}

func assertAggregateEqual(t *testing.T, act *salsa.Aggregate[state], exp aggregate) {
	assertDeepEqual(t, aggregate{
		initState: act.InitState(),
		currState: act.State(),
		events:    act.Events(),
	}, exp)
}
