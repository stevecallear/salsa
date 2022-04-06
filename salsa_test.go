package salsa_test

import (
	"errors"
	"reflect"
	"testing"
)

type (
	state struct {
		Balance int `json:"balance"`
	}

	event struct {
		Amount int `json:"amount"`
	}

	errEvent struct{}
)

func (e *event) Type() string {
	return "event"
}

func (e *event) Apply(s state) (state, error) {
	s.Balance += e.Amount
	return s, nil
}

func (e *errEvent) Type() string {
	return "errevent"
}

func (e *errEvent) Apply(s state) (state, error) {
	return s, errors.New("error")
}

func assertErrorExists(t *testing.T, act error, exp bool) {
	if act != nil && !exp {
		t.Errorf("got %v, expected nil", act)
	}
	if act == nil && exp {
		t.Error("got nil, expected an error")
	}
}

func assertDeepEqual(t *testing.T, act, exp interface{}) {
	if !reflect.DeepEqual(act, exp) {
		t.Errorf("got %v, expected %v", act, exp)
	}
}
