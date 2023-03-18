package infra

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"time"

	. "github.com/infrago/base"
)

var (
	ErrInvalidData = errors.New("Invalid data.")
)

func builtin() {

	gob.Register(time.Now())
	gob.Register(Map{})
	gob.Register([]Map{})
	gob.Register([]Any{})

	Register(JSON, Codec{
		Name: "内置JSON编解码", Text: "内置JSON编解码",
		Encode: func(value Any) (Any, error) {
			return json.Marshal(value)
		},
		Decode: func(data Any, value Any) (Any, error) {
			if bytes, ok := data.([]byte); ok {
				err := json.Unmarshal(bytes, value)
				if err != nil {
					return nil, err
				}
				return value, nil
			}
			return nil, ErrInvalidData
		},
	}, false)

	Register(GOB, Codec{
		Name: "内置GOB编解码", Text: "内置GOB编解码",
		Encode: func(value Any) (Any, error) {
			var buffer bytes.Buffer
			encoder := gob.NewEncoder(&buffer)
			err := encoder.Encode(value)
			if err != nil {
				return nil, err
			}
			return buffer.Bytes(), nil
		},
		Decode: func(data Any, value Any) (Any, error) {
			if dataBytes, ok := data.([]byte); ok {
				buffer := bytes.NewReader(dataBytes)
				decoder := gob.NewDecoder(buffer)
				err := decoder.Decode(value)
				if err != nil {
					return nil, err
				}

				return value, nil
			}
			return nil, ErrInvalidData
		},
	}, false)
}
