package salsa_test

import (
	"testing"

	"github.com/stevecallear/salsa"
)

func TestEncodeJSON(t *testing.T) {
	var b []byte
	var err error
	exp := state{Balance: 10}

	t.Run("should encode the value", func(t *testing.T) {
		b, err = salsa.EncodeJSON.Encode(&exp)
		assertErrorExists(t, err, false)
	})

	t.Run("should decode the value", func(t *testing.T) {
		var act state
		err = salsa.DecodeJSON.Decode(b, &act)
		assertErrorExists(t, err, false)
		assertDeepEqual(t, act, exp)
	})
}
