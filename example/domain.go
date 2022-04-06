package main

import (
	"errors"

	"github.com/stevecallear/salsa"
)

type (
	AccountState struct {
		ID      string `json:"id"`
		Balance int64  `json:"balance"`
	}

	CreateAccountEvent struct {
		ID string `json:"id"`
	}

	CreditAccountEvent struct {
		Amount int64 `json:"amount"`
	}
)

var ResolveAccountEvent = salsa.EventResolverFunc[AccountState](func(eventType string) (salsa.Event[AccountState], error) {
	switch eventType {
	case new(CreateAccountEvent).Type():
		return new(CreateAccountEvent), nil
	case new(CreditAccountEvent).Type():
		return new(CreditAccountEvent), nil
	default:
		return nil, errors.New("invalid event type")
	}
})

func (e *CreateAccountEvent) Type() string {
	return "account.create"
}

func (e *CreateAccountEvent) Apply(s AccountState) (AccountState, error) {
	if e.ID == "" {
		return s, errors.New("account id is invalid")
	}

	if s.ID != "" {
		return s, errors.New("account has already been created")
	}

	s.ID = e.ID
	return s, nil
}

func (e *CreditAccountEvent) Type() string {
	return "account.credit"
}

func (e *CreditAccountEvent) Apply(s AccountState) (AccountState, error) {
	if e.Amount <= 0 {
		return s, errors.New("credit amount must be greater than zero")
	}

	s.Balance += e.Amount
	return s, nil
}
