package main

import (
	"context"

	"github.com/stevecallear/salsa"
)

type AccountService struct {
	store *salsa.Store[string, AccountState]
}

func NewAccountService(s *salsa.Store[string, AccountState]) *AccountService {
	return &AccountService{store: s}
}

func (s *AccountService) GetAccount(ctx context.Context, id string) (AccountState, error) {
	a, err := s.store.Get(ctx, id)
	if err != nil {
		return AccountState{}, err
	}

	return a.State(), nil
}

func (s *AccountService) CreateAccount(ctx context.Context, id string, balance int64) error {
	a := new(salsa.Aggregate[AccountState])

	for _, e := range []salsa.Event[AccountState]{
		&CreateAccountEvent{ID: id},
		&CreditAccountEvent{Amount: balance},
	} {
		if _, err := a.Apply(e); err != nil {
			return err
		}
	}

	return s.store.Save(ctx, id, a)
}

func (s *AccountService) CreditAccount(ctx context.Context, id string, amount int64) error {
	a, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}

	if _, err := a.Apply(&CreditAccountEvent{Amount: amount}); err != nil {
		return err
	}

	return s.store.Save(ctx, id, a)
}
