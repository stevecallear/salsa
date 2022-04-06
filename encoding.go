package salsa

import "encoding/json"

type (
	// Encoder represents an encoder
	Encoder interface {
		Encode(v any) ([]byte, error)
	}

	// EncoderFunc represents an encoder func
	EncoderFunc func(v any) ([]byte, error)

	// Decoder represents a decoder
	Decoder interface {
		Decode(b []byte, v any) error
	}

	// DecoderFunc represents a decoder func
	DecoderFunc func(b []byte, v any) error
)

var (
	// EncodeJSON encodes the specified value as JSON
	EncodeJSON EncoderFunc = func(v any) ([]byte, error) {
		return json.Marshal(v)
	}

	// DecodeJSON decodes the specified JSON into the value
	DecodeJSON DecoderFunc = func(b []byte, v any) error {
		return json.Unmarshal(b, v)
	}
)

// Encode encodes the value
func (e EncoderFunc) Encode(v any) ([]byte, error) {
	return e(v)
}

// Decode decodes the value
func (d DecoderFunc) Decode(b []byte, v any) error {
	return d(b, v)
}
