package dynamo

import (
	"context"
	"errors"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/stevecallear/salsa"
)

type (
	db struct {
		tableName string
		client    *dynamodb.Client
	}

	tx struct {
		tableName string
		id        string
		input     *dynamodb.TransactWriteItemsInput
	}
)

const stateType = "STATE"

// CreateTable creates the required dynamodb table for the event store
func CreateTable(ctx context.Context, c *dynamodb.Client, tableName string) error {
	_, err := c.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("version"),
				AttributeType: types.ScalarAttributeTypeN,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("version"),
				KeyType:       types.KeyTypeRange,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})

	if err != nil {
		var terr *types.ResourceInUseException
		if errors.As(err, &terr) {
			err = nil
		}
	}

	return err
}

// New returns a new event store backed by dynamodb
func New[T any](c *dynamodb.Client, tableName string, optFns ...func(*salsa.Options[T])) *salsa.Store[string, T] {
	return salsa.NewStore[string](&db{
		tableName: tableName,
		client:    c,
	}, optFns...)
}

// Read reads most recent state and events for the specified id
func (d *db) Read(ctx context.Context, id string, limit int) (salsa.EncodedState, []salsa.EncodedEvent, error) {
	res, err := d.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(d.tableName),
		KeyConditionExpression: aws.String("#i = :i"),
		ExpressionAttributeNames: map[string]string{
			"#i": "id",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":i": &types.AttributeValueMemberS{Value: id},
		},
		ScanIndexForward: aws.Bool(false),
		ConsistentRead:   aws.Bool(true),
		Limit:            aws.Int32(int32(limit)),
	})
	if err != nil {
		return salsa.EncodedState{}, nil, err
	}

	if len(res.Items) < 1 {
		return salsa.EncodedState{}, nil, errors.New("not found")
	}

	var state salsa.EncodedState
	var events []salsa.EncodedEvent

	for _, itm := range res.Items {
		if d.avIsState(itm) {
			state, err = d.avToState(itm)
			if err != nil {
				return salsa.EncodedState{}, nil, err
			}
			break
		} else {
			var e salsa.EncodedEvent
			if e, err = d.avToEvent(itm); err != nil {
				return salsa.EncodedState{}, nil, err
			}
			events = append(events, e)
		}
	}

	reverse(events)
	return state, events, nil
}

// Write executes the specified write function within a transaction
func (d *db) Write(ctx context.Context, id string, fn func(salsa.DBTx) error) error {
	in := &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{},
	}

	t := &tx{
		tableName: d.tableName,
		id:        id,
		input:     in,
	}

	if err := fn(t); err != nil {
		return err
	}

	_, err := d.client.TransactWriteItems(ctx, in)
	return err
}

func (s *db) avIsState(av map[string]types.AttributeValue) bool {
	if v, ok := av["type"].(*types.AttributeValueMemberS); ok {
		return v.Value == stateType
	}

	return false
}

func (db *db) avToState(av map[string]types.AttributeValue) (salsa.EncodedState, error) {
	var vs salsa.EncodedState
	var err error

	vm := av["version"].(*types.AttributeValueMemberN).Value
	vs.Version, err = strconv.ParseUint(vm, 10, 64)
	if err != nil {
		return vs, err
	}

	dm := av["data"].(*types.AttributeValueMemberS).Value
	vs.Data = []byte(dm)

	return vs, nil
}

func (db *db) avToEvent(av map[string]types.AttributeValue) (salsa.EncodedEvent, error) {
	var e salsa.EncodedEvent
	var err error

	e.Type = av["type"].(*types.AttributeValueMemberS).Value

	vm := av["version"].(*types.AttributeValueMemberN).Value
	e.Version, err = strconv.ParseUint(vm, 10, 64)
	if err != nil {
		return e, err
	}

	dm := av["data"].(*types.AttributeValueMemberS).Value
	e.Data = []byte(dm)

	return e, nil
}

// Event writes the specified event
func (t *tx) Event(e salsa.EncodedEvent) error {
	t.append(t.eventToAV(e))
	return nil
}

// State writes the specified state
func (t *tx) State(s salsa.EncodedState) error {
	t.append(t.stateToAV(s))
	return nil
}

func (t *tx) append(av map[string]types.AttributeValue) {
	t.input.TransactItems = append(t.input.TransactItems, types.TransactWriteItem{
		Put: &types.Put{
			TableName:           aws.String(t.tableName),
			ConditionExpression: aws.String("(attribute_not_exists (#i)) AND (attribute_not_exists (#v))"),
			ExpressionAttributeNames: map[string]string{
				"#i": "id",
				"#v": "version",
			},
			Item: av,
		},
	})
}

func (t *tx) stateToAV(s salsa.EncodedState) map[string]types.AttributeValue {
	ve := strconv.FormatUint(s.Version, 10)
	return map[string]types.AttributeValue{
		"id":      &types.AttributeValueMemberS{Value: t.id},
		"version": &types.AttributeValueMemberN{Value: ve},
		"type":    &types.AttributeValueMemberS{Value: stateType},
		"data":    &types.AttributeValueMemberS{Value: string(s.Data)},
	}
}

func (t *tx) eventToAV(e salsa.EncodedEvent) map[string]types.AttributeValue {
	ve := strconv.FormatUint(e.Version, 10)
	return map[string]types.AttributeValue{
		"id":      &types.AttributeValueMemberS{Value: t.id},
		"version": &types.AttributeValueMemberN{Value: ve},
		"type":    &types.AttributeValueMemberS{Value: e.Type},
		"data":    &types.AttributeValueMemberS{Value: string(e.Data)},
	}
}

func reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
