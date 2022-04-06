# salsa
[![Build Status](https://github.com/stevecallear/salsa/actions/workflows/build.yml/badge.svg)](https://github.com/stevecallear/salsa/actions/workflows/build.yml)
[![codecov](https://codecov.io/gh/stevecallear/salsa/branch/master/graph/badge.svg)](https://codecov.io/gh/stevecallear/salsa)
[![Go Report Card](https://goreportcard.com/badge/github.com/stevecallear/salsa)](https://goreportcard.com/report/github.com/stevecallear/salsa)

`salsa` is a basic event sourcing implementation using Go 1.18 generics. It was built as part of a more general CQRS/DDD Lite project and while it is not intended for production use it contains some concepts that may be useful to others who are exploring similar approaches.

## Getting Started

```
go get github.com/stevecallear/salsa@latest
```

See the [example](#) for a basic implementation using an in-memory event store.

## Aggregate

`salsa.Aggregate[T]` specifies and aggregate with state of type `T`. Aggregate state is updated by supplying a `salsa.Event[T]` implementation to the `Apply` method.

The implementation assumes that all business logic is implemented in the domain events, so leans towards the anemic domain approach. While this would typically be considered an anti-pattern the use of event sourcing ensures that logic is applied in a consistent manner. To simplify the contract an aggregate wrapper could be created the builds and applies the correct events, or alternatively an application service could be used as per the example.

## Store

`salsa.Store[TID, TState]` provides an event store implementation that encodes/decodes events and snapshot state and persists them to the supplied backing store.

`salsa.NewMemoryStore()` returns a store backed by an in-memory implementation. Other backing stores can be configured by implementing `salsa.DB[TID]` See the in-memory implementation for an example of how to create alternative backing stores.

### Event Resolution

To ensure that events can be correctly decoded, a `salsa.EventResolver[T]` implementation must be provided when creating the store. By default an error will be returned for all event types.

```
func resolveEvent(eventType string) (salsa.Event[state], error) {
	switch eventType {
	case new(event).Type():
		return new(event), nil
	default:
		return nil, errors.New("invalid event type")
	}
}

...

s := salsa.NewStore(db, salsa.WithResolver[state](salsa.EventResolverFunc[state](resolveEvent)))
```

### Snapshot Rate

The store uses snapshots to reduce the amount of data that needs to be retrieved to build an aggregate. By default a snapshot is persisted every 10 events, but this value can be configured as part of the store options.

```
s := salsa.NewStore(db, salsa.WithSnapshotRate[state](100))
```

> Note: the store builds snapshots by replaying all events from the initial aggregate state. As a result, events must operate only on the supplied state.

### Encoding

Persisted events and state are encoded using the supplied `Encoder[T]` and `Decoder[T]` implementations. JSON encoding/decoding is used by default, with alternative implementations being configured as part of the store options.

For example GOB encoding could be configured using the following:

```
s := salsa.NewStore(db, func(o *salsa.Options[state]) {
    o.Encoder = salsa.EncoderFunc(func(v any) ([]byte, error) {
        b := bytes.NewBuffer(nil)
        if err := gob.NewEncoder(b).Encode(v); err != nil {
            return nil, err
        }
        return b.Bytes(), nil
    })
    o.Decoder = salsa.DecoderFunc(func(b []byte, v any) error {
        buf := bytes.NewBuffer(b)
        return gob.NewDecoder(buf).Decode(v)
    })
})
```
