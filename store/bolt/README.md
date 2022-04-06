# bolt

`bolt` provides a BoltDB backing store implementation for `salsa`.

## Getting Started

```
go get github.com/stevecallear/salsa/store/bolt@latest
```

```
db, err := bbolt.Open("eventstore.db", 0666, nil)
if err != nil {
    log.Fatal(err)
}
defer db.Close()

s := bolt.New(db, salsa.WithResolver[state](salsa.EventResolverFunc[state](resolveEvent)))
```
