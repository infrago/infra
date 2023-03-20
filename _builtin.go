package infra

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"strings"
	"time"

	. "github.com/infrago/base"
	"github.com/infrago/util"
)

var (
	ErrInvalidData = errors.New("Invalid data.")
)
var (
	textCoder *base64.Encoding
)

func builtin() {
	textCoder = base64.NewEncoding(TextAlphabet())

	codecBuiltin()
}

func codecBuiltin() {

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

	Register(XML, Codec{
		Name: "XML编解码", Text: "XML编解码",
		Encode: func(value Any) (Any, error) {
			return xml.Marshal(value)
		},
		Decode: func(data Any, value Any) (Any, error) {
			if dataBytes, ok := data.([]byte); ok {
				err := xml.Unmarshal(dataBytes, value)
				if err != nil {
					return nil, err
				}
				return value, nil
			}
			return nil, errInvalidData
		},
	}, false)

	Register(TEXT, Codec{
		Name: "文本加密", Text: "文本加密，自定义字符表的base64编码，字典：" + TextAlphabet(),
		Encode: func(value Any) (Any, error) {
			var bytes []byte
			if vvs, ok := value.([]byte); ok {
				bytes = vvs
			} else {
				bytes = []byte(util.AnyToString(value))
			}

			text := textCoder.EncodeToString(bytes)
			return text, nil
		},
		Decode: func(data Any, value Any) (Any, error) {
			var text string
			if vvs, ok := value.(string); ok {
				text = vvs
			} else {
				text = util.AnyToString(data)
			}
			return textCoder.DecodeString(text)
		},
	}, false)
	Register(TEXTS, Codec{
		Name: "文本数组加密", Text: "文本数组加密，自定义字符表的base64编码，字典：" + TextAlphabet(),
		Encode: func(value Any) (Any, error) {
			text := ""
			if vvs, ok := value.(string); ok {
				text = vvs
			} else if vvs, ok := value.([]string); ok {
				text = strings.Join(vvs, "\n")
			} else {
				text = util.AnyToString(value)
			}
			return textCoder.EncodeToString([]byte(text)), nil
		},
		Decode: func(data Any, value Any) (Any, error) {
			text := util.AnyToString(data)
			bytes, err := textCoder.DecodeString(text)
			if err != nil {
				return nil, err
			}
			return strings.Split(string(bytes), "\n"), nil
		},
	}, false)
}
