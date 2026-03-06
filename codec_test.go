package infra

import (
	"errors"
	"testing"

	. "github.com/infrago/base"
)

func TestEncryptRejectsNonStringCodecOutput(t *testing.T) {
	name := "codec_test_non_string_output"
	Register(name, Codec{
		Encode: func(v Any) (Any, error) {
			return 123, nil
		},
		Decode: func(d Any, v Any) (Any, error) {
			return d, nil
		},
	})

	if _, err := Encrypt(name, "demo"); !errors.Is(err, ErrInvalidCodecData) {
		t.Fatalf("expected ErrInvalidCodecData, got %v", err)
	}
}
