# dynamo

`dynamo` provides a DynamoDB backing store implementation for `salsa`.

## Getting Started

```
go get github.com/stevecallear/salsa/store/dynamo@latest
```

```
cfg, err := config.LoadDefaultConfig(context.Background())
if err != nil {
    log.Fatal(err)
}

client := dynamodb.NewFromConfig(cfg)

if err := dynamo.CreateTable(context.Background(), client, "table-name"); err != nil {
    log.Fatal(err)
}

s := dynamo.New(client, "table-name",
    salsa.WithResolver[state](salsa.EventResolverFunc[state](resolveEvent)))
```
