package salsa_test

import (
	"errors"
	"testing"

	"github.com/stevecallear/salsa"
)

func TestEventResolver_Resolve(t *testing.T) {
	evt := new(event)

	tests := []struct {
		name string
		sut  salsa.EventResolverFunc[state]
		exp  salsa.Event[state]
		err  bool
	}{
		{
			name: "should return func errors",
			sut: salsa.EventResolverFunc[state](func(t string) (salsa.Event[state], error) {
				return nil, errors.New("error")
			}),
			err: true,
		},
		{
			name: "should return the resolved event",
			sut: salsa.EventResolverFunc[state](func(t string) (salsa.Event[state], error) {
				return evt, nil
			}),
			exp: evt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			act, err := tt.sut.Resolve("type")
			assertErrorExists(t, err, tt.err)
			assertDeepEqual(t, act, tt.exp)
		})
	}
}
