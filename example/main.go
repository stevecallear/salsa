package main

import (
	"context"
	"fmt"
	"log"

	"github.com/stevecallear/salsa"
)

func main() {
	str := salsa.NewMemoryStore[string](salsa.WithResolver[AccountState](ResolveAccountEvent))
	svc := NewAccountService(str)

	const id = "accountid"
	ctx := context.Background()

	if err := svc.CreateAccount(ctx, id, 100); err != nil {
		log.Fatal(err)
	}

	if err := svc.CreditAccount(ctx, id, 50); err != nil {
		log.Fatal(err)
	}

	a, err := svc.GetAccount(ctx, id)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(a.Balance)
	// Output: 150
}
