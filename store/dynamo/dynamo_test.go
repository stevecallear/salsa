package dynamo_test

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"github.com/stevecallear/salsa"
	"github.com/stevecallear/salsa/store/dynamo"
)

func TestMain(m *testing.M) {
	client = newLocalClient()
	for _, tn := range []string{testCreateTableName, testNewName} {
		_, err := client.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
			TableName: aws.String(tn),
		})
		if err != nil {
			var terr *types.ResourceNotFoundException
			if !errors.As(err, &terr) {
				panic(err)
			}
		}
	}

	m.Run()
}

const (
	testCreateTableName = "salsa-testcreatetable"
	testNewName         = "salsa-testnew"
)

var client *dynamodb.Client

func TestCreateTable(t *testing.T) {
	t.Run("should create the table", func(t *testing.T) {
		err := dynamo.CreateTable(context.Background(), client, testCreateTableName)
		assertErrorExists(t, err, false)

		res, err := client.ListTables(context.Background(), new(dynamodb.ListTablesInput))
		assertErrorExists(t, err, false)

		var ok bool
		for _, tn := range res.TableNames {
			if tn == testCreateTableName {
				ok = true
				break
			}
		}

		if act, exp := ok, true; act != exp {
			t.Errorf("got %v, expected %v", act, exp)
		}
	})

	t.Run("should not return an error if the table exists", func(t *testing.T) {
		err := dynamo.CreateTable(context.Background(), client, testCreateTableName)
		assertErrorExists(t, err, false)
	})
}

func TestNew(t *testing.T) {
	if err := dynamo.CreateTable(context.Background(), client, testNewName); err != nil {
		t.Fatal(err)
	}

	er := salsa.EventResolverFunc[state](func(string) (salsa.Event[state], error) {
		return new(event), nil
	})

	sut := dynamo.New(client, testNewName, salsa.WithResolver[state](er), salsa.WithSnapshotRate[state](5))

	id := uuid.NewString()
	t.Run("should return an error if the aggregate does not exist", func(t *testing.T) {
		_, err := sut.Get(context.Background(), id)
		assertErrorExists(t, err, true)
	})

	t.Run("should write the aggregate", func(t *testing.T) {
		a := new(salsa.Aggregate[state])
		for i := 1; i <= 12; i++ {
			_, err := a.Apply(&event{Amount: i * 10})
			assertErrorExists(t, err, false)
		}

		err := sut.Save(context.Background(), id, a)
		assertErrorExists(t, err, false)
	})

	t.Run("should return an error if a conflict occurs", func(t *testing.T) {
		a := new(salsa.Aggregate[state])
		_, err := a.Apply(&event{Amount: 100})
		assertErrorExists(t, err, false)

		err = sut.Save(context.Background(), id, a)
		assertErrorExists(t, err, true)
	})

	t.Run("should read the aggregate (state)", func(t *testing.T) {
		act, err := sut.Get(context.Background(), id)
		assertErrorExists(t, err, false)

		assertAggregateEqual(t, act, aggregate{
			state: state{Balance: 780},
			versions: salsa.Versions{
				State:   12,
				Initial: 12,
				Current: 12,
			},
		})
	})

	t.Run("should write additional events", func(t *testing.T) {
		a, err := sut.Get(context.Background(), id)
		assertErrorExists(t, err, false)

		for i := 1; i <= 2; i++ {
			_, err := a.Apply(&event{Amount: i * 10})
			assertErrorExists(t, err, false)
		}

		err = sut.Save(context.Background(), id, a)
		assertErrorExists(t, err, false)
	})

	t.Run("should read the aggregate (state and events)", func(t *testing.T) {
		act, err := sut.Get(context.Background(), id)
		assertErrorExists(t, err, false)

		assertAggregateEqual(t, act, aggregate{
			state: state{Balance: 810},
			versions: salsa.Versions{
				State:   12,
				Initial: 14,
				Current: 14,
			},
		})
	})
}

func newLocalClient() *dynamodb.Client {
	ep := os.Getenv("DYNAMO_ENDPOINT_URL")
	if ep == "" {
		ep = "http://db:8000"
	}

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("eu-west-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("accesskeyid", "secretkey", "")))
	if err != nil {
		panic(err)
	}

	return dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.EndpointResolver = dynamodb.EndpointResolverFromURL(ep)
	})
}

type (
	state struct {
		Balance int `json:"balance"`
	}

	event struct {
		Amount int `json:"amount"`
	}

	aggregate struct {
		state    state
		versions salsa.Versions
		events   []salsa.Event[state]
	}
)

func (e *event) Type() string {
	return "event"
}

func (e *event) Apply(s state) (state, error) {
	s.Balance += e.Amount
	return s, nil
}

func assertErrorExists(t *testing.T, act error, exp bool) {
	if act != nil && !exp {
		t.Errorf("got %v, expected nil", act)
	}
	if act == nil && exp {
		t.Error("got nil, expected an error")
	}
}

func assertAggregateEqual(t *testing.T, act *salsa.Aggregate[state], exp aggregate) {
	a := aggregate{
		state:    act.State(),
		versions: act.Versions(),
		events:   act.Events(),
	}

	if !reflect.DeepEqual(a, exp) {
		t.Errorf("got %v, expected %v", act, exp)
	}
}
