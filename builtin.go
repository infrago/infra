package infra

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"
	"time"

	. "github.com/infrago/base"
	"github.com/pelletier/go-toml/v2"
)

var builtin = &builtinModule{}

type builtinModule struct {
	loaded bool
}

func (m *builtinModule) Register(string, Any) {}
func (m *builtinModule) Config(Map)           {}
func (m *builtinModule) Open()                {}
func (m *builtinModule) Start()               {}
func (m *builtinModule) Stop()                {}
func (m *builtinModule) Close()               {}

func (m *builtinModule) Setup() {
	if m.loaded {
		return
	}

	// builtin defaults should never block project-level overrides.
	origin := Override()
	Override(false)
	defer Override(origin)

	registerBuiltinCodecs()
	registerBuiltinMimes()
	registerBuiltinRegulars()
	registerBuiltinTypes()

	m.loaded = true
}

func registerBuiltinCodecs() {
	codec.RegisterCodec(JSON, Codec{
		Encode: func(v Any) (Any, error) {
			if bts, ok := v.([]byte); ok {
				return bts, nil
			}
			return json.Marshal(v)
		},
		Decode: func(d Any, v Any) (Any, error) {
			data, ok := builtinCodecToBytes(d)
			if !ok {
				return nil, errInvalidCodecData
			}
			if v != nil {
				return v, json.Unmarshal(data, v)
			}
			var out Any
			return out, json.Unmarshal(data, &out)
		},
	})
	codec.RegisterCodec(XML, Codec{
		Encode: func(v Any) (Any, error) {
			if bts, ok := v.([]byte); ok {
				return bts, nil
			}
			return xml.Marshal(v)
		},
		Decode: func(d Any, v Any) (Any, error) {
			data, ok := builtinCodecToBytes(d)
			if !ok {
				return nil, errInvalidCodecData
			}
			if v != nil {
				return v, xml.Unmarshal(data, v)
			}
			var out Any
			return out, xml.Unmarshal(data, &out)
		},
	})
	codec.RegisterCodec(GOB, Codec{
		Encode: func(v Any) (Any, error) {
			if bts, ok := v.([]byte); ok {
				return bts, nil
			}
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			if err := enc.Encode(v); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		},
		Decode: func(d Any, v Any) (Any, error) {
			data, ok := builtinCodecToBytes(d)
			if !ok {
				return nil, errInvalidCodecData
			}
			if v == nil {
				var out Any
				v = &out
			}
			dec := gob.NewDecoder(bytes.NewReader(data))
			return v, dec.Decode(v)
		},
	})
	codec.RegisterCodec(TOML, Codec{
		Encode: func(v Any) (Any, error) {
			if bts, ok := v.([]byte); ok {
				return bts, nil
			}
			return toml.Marshal(v)
		},
		Decode: func(d Any, v Any) (Any, error) {
			data, ok := builtinCodecToBytes(d)
			if !ok {
				return nil, errInvalidCodecData
			}
			if v != nil {
				return v, toml.Unmarshal(data, v)
			}
			var out Any
			return out, toml.Unmarshal(data, &out)
		},
	})
	codec.RegisterCodec(DIGIT, Codec{
		Encode: func(v Any) (Any, error) {
			n, ok := builtinToInt64(v)
			if !ok {
				return nil, errInvalidCodecData
			}
			return encodeInt64(n, codec.config.Digit, codec.config.Salt, codec.config.Length)
		},
		Decode: func(d Any, v Any) (Any, error) {
			s := builtinCodecToString(d)
			if s == "" {
				return nil, errInvalidCodecData
			}
			return decodeInt64(s, codec.config.Digit, codec.config.Salt)
		},
	})
	codec.RegisterCodec(DIGITS, Codec{
		Encode: func(v Any) (Any, error) {
			arr, err := builtinCodecToInt64Slice(v)
			if err != nil {
				return nil, err
			}
			return encodeInt64Slice(arr, codec.config.Digit, codec.config.Salt, codec.config.Length)
		},
		Decode: func(d Any, v Any) (Any, error) {
			s := builtinCodecToString(d)
			if s == "" {
				return nil, errInvalidCodecData
			}
			return decodeInt64Slice(s, codec.config.Digit, codec.config.Salt)
		},
	})
	codec.RegisterCodec(TEXT, Codec{
		Encode: func(v Any) (Any, error) {
			var data []byte
			switch vv := v.(type) {
			case []byte:
				data = vv
			case string:
				data = []byte(vv)
			default:
				bts, err := json.Marshal(v)
				if err != nil {
					return nil, err
				}
				data = bts
			}
			return encodeBytes(data, codec.config.Text, codec.config.Salt)
		},
		Decode: func(d Any, v Any) (Any, error) {
			s := builtinCodecToString(d)
			if s == "" {
				return nil, errInvalidCodecData
			}
			data, err := decodeBytes(s, codec.config.Text, codec.config.Salt)
			if err != nil {
				return nil, err
			}
			if v != nil {
				return v, json.Unmarshal(data, v)
			}
			return data, nil
		},
	})
	codec.RegisterCodec(TEXTS, Codec{
		Encode: func(v Any) (Any, error) {
			arr := builtinToStringSlice(v)
			bts, err := json.Marshal(arr)
			if err != nil {
				return nil, err
			}
			return encodeBytes(bts, codec.config.Text, codec.config.Salt)
		},
		Decode: func(d Any, v Any) (Any, error) {
			s := builtinCodecToString(d)
			if s == "" {
				return nil, errInvalidCodecData
			}
			data, err := decodeBytes(s, codec.config.Text, codec.config.Salt)
			if err != nil {
				return nil, err
			}
			var out []string
			if err := json.Unmarshal(data, &out); err != nil {
				return nil, err
			}
			return out, nil
		},
	})

	codec.RegisterCodec("base64", Codec{
		Alias: []string{"base64std"},
		Encode: func(v Any) (Any, error) {
			return base64.StdEncoding.EncodeToString([]byte(builtinToText(v))), nil
		},
		Decode: func(d Any, v Any) (Any, error) {
			out, err := base64.StdEncoding.DecodeString(builtinToText(d))
			if err != nil {
				return nil, err
			}
			return string(out), nil
		},
	})

	codec.RegisterCodec("base64url", Codec{
		Encode: func(v Any) (Any, error) {
			return base64.URLEncoding.EncodeToString([]byte(builtinToText(v))), nil
		},
		Decode: func(d Any, v Any) (Any, error) {
			out, err := base64.URLEncoding.DecodeString(builtinToText(d))
			if err != nil {
				return nil, err
			}
			return string(out), nil
		},
	})
}

func builtinCodecToBytes(v Any) ([]byte, bool) {
	switch vv := v.(type) {
	case []byte:
		return vv, true
	case string:
		return []byte(vv), true
	default:
		return nil, false
	}
}

func builtinCodecToString(v Any) string {
	switch vv := v.(type) {
	case string:
		return vv
	case []byte:
		return string(vv)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func builtinCodecToInt64Slice(v Any) ([]int64, error) {
	switch vv := v.(type) {
	case []int64:
		return vv, nil
	case []int:
		out := make([]int64, 0, len(vv))
		for _, n := range vv {
			out = append(out, int64(n))
		}
		return out, nil
	case []Any:
		out := make([]int64, 0, len(vv))
		for _, n := range vv {
			val, ok := builtinToInt64(n)
			if !ok {
				return nil, errInvalidCodecData
			}
			out = append(out, val)
		}
		return out, nil
	default:
		return nil, errInvalidCodecData
	}
}

func registerBuiltinMimes() {
	basic.RegisterMimes(Mimes{
		"text":   {"text/plain"},
		"html":   {"text/html"},
		"xml":    {"application/xml"},
		"json":   {"application/json"},
		"file":   {"application/octet-stream"},
		"down":   {"application/octet-stream"},
		"script": {"text/html"},
		"view":   {"text/html"},
		"css":    {"text/css"},
		"js":     {"application/javascript"},
		"txt":    {"text/plain"},
		"csv":    {"text/csv"},
		"md":     {"text/markdown"},
		"pdf":    {"application/pdf"},
		"zip":    {"application/zip"},
		"gz":     {"application/x-gzip"},
		"tar":    {"application/x-tar"},
		"rar":    {"application/vnd.rar"},
		"7z":     {"application/x-7z-compressed"},
		"doc":    {"application/msword"},
		"docx":   {"application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		"xls":    {"application/vnd.ms-excel"},
		"xlsx":   {"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		"ppt":    {"application/vnd.ms-powerpoint"},
		"pptx":   {"application/vnd.openxmlformats-officedocument.presentationml.presentation"},
		"jpg":    {"image/jpeg"},
		"jpeg":   {"image/jpeg"},
		"png":    {"image/png"},
		"gif":    {"image/gif"},
		"webp":   {"image/webp"},
		"svg":    {"image/svg+xml"},
		"ico":    {"image/x-icon"},
		"bmp":    {"image/bmp"},
		"mp3":    {"audio/mpeg"},
		"wav":    {"audio/x-wav"},
		"aac":    {"audio/aac"},
		"flac":   {"audio/flac"},
		"m4a":    {"audio/mp4"},
		"ogg":    {"audio/ogg"},
		"mp4":    {"video/mp4"},
		"mov":    {"video/quicktime"},
		"avi":    {"video/x-msvideo"},
		"mpeg":   {"video/mpeg"},
		"mpg":    {"video/mpeg"},
		"m3u8":   {"application/vnd.apple.mpegurl"},
		"ts":     {"video/mp2t"},
		"webm":   {"video/webm"},
		"apk":    {"application/vnd.android.package-archive"},
		"ipa":    {"application/vnd.iphone"},
		"*":      {"application/octet-stream"},
		"":       {"application/octet-stream"},
	})
}

func registerBuiltinRegulars() {
	basic.RegisterRegulars(Regulars{
		"password": {`^[0-9A-Fa-f]{40}$`},
		"number":   {`^[0-9]+$`},
		"float":    {`^[+-]?([0-9]+(\.[0-9]+)?|\.[0-9]+)$`},
		"date": {
			`^(\d{4})(\d{2})(\d{2})$`,
			`^(\d{4})-(\d{2})-(\d{2})$`,
			`^(\d{10,15})$`,
		},
		"datetime": {
			`^(\d{4})-(\d{2})-(\d{2})$`,
			`^(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2}):(\d{2})$`,
			`^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})Z$`,
			`^(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2}):(\d{2})\.(\d{3})$`,
			`^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})\.(\d{3})Z$`,
			`^(\d{10,15})$`,
		},
		"mobile": {`^1[0-9]{10}$`},
		"idcard": {`(^\d{15}$)|(^\d{18}$)|(^\d{17}(\d|X|x)$)`},
		"email":  {`^[0-9A-Za-z][_.0-9A-Za-z-]{0,31}@([0-9A-Za-z][0-9A-Za-z-]{0,30}[0-9A-Za-z]\.){1,4}[A-Za-z]{2,20}$`},
	})
}

func registerBuiltinTypes() {
	// password
	basic.RegisterType("password", Type{
		Name: "password",
		Valid: func(value Any, config Var) bool {
			return builtinToText(value) != ""
		},
		Value: func(value Any, config Var) Any {
			text := builtinToText(value)
			if Match("password", text) {
				return text
			}
			sum := sha1.Sum([]byte(text))
			return hex.EncodeToString(sum[:])
		},
	})

	// any/map
	basic.RegisterType("any", Type{
		Name:  "any",
		Alias: []string{"*"},
		Valid: func(value Any, config Var) bool { return true },
		Value: func(value Any, config Var) Any { return value },
	})
	basic.RegisterType("[any]", Type{
		Name:  "[any]",
		Alias: []string{"anys"},
		Valid: func(value Any, config Var) bool { return true },
		Value: func(value Any, config Var) Any {
			switch v := value.(type) {
			case []Any:
				return v
			case []string:
				out := make([]Any, 0, len(v))
				for _, one := range v {
					out = append(out, one)
				}
				return out
			default:
				return []Any{v}
			}
		},
	})

	basic.RegisterType("map", Type{
		Name:  "map",
		Alias: []string{"object", "dict"},
		Valid: func(value Any, config Var) bool {
			switch value.(type) {
			case Map, []Map:
				return true
			}
			return false
		},
		Value: func(value Any, config Var) Any {
			switch v := value.(type) {
			case Map:
				return v
			case []Map:
				if len(v) > 0 {
					return v[0]
				}
			}
			return Map{}
		},
	})
	basic.RegisterType("[map]", Type{
		Name:  "[map]",
		Alias: []string{"array_map", "maps"},
		Valid: func(value Any, config Var) bool {
			switch value.(type) {
			case Map, []Map:
				return true
			}
			return false
		},
		Value: func(value Any, config Var) Any {
			switch v := value.(type) {
			case Map:
				return []Map{v}
			case []Map:
				return v
			}
			return []Map{}
		},
	})

	// bool
	basic.RegisterType("bool", Type{
		Name: "bool",
		Valid: func(value Any, config Var) bool {
			_, ok := builtinToBool(value)
			return ok
		},
		Value: func(value Any, config Var) Any {
			val, _ := builtinToBool(value)
			return val
		},
	})
	basic.RegisterType("[bool]", Type{
		Name: "[bool]",
		Valid: func(value Any, config Var) bool {
			switch v := value.(type) {
			case []bool:
				return true
			case []Any:
				for _, one := range v {
					if _, ok := builtinToBool(one); !ok {
						return false
					}
				}
				return true
			default:
				_, ok := builtinToBool(value)
				return ok
			}
		},
		Value: func(value Any, config Var) Any {
			switch v := value.(type) {
			case []bool:
				return v
			case []Any:
				out := make([]bool, 0, len(v))
				for _, one := range v {
					bv, _ := builtinToBool(one)
					out = append(out, bv)
				}
				return out
			default:
				bv, _ := builtinToBool(value)
				return []bool{bv}
			}
		},
	})

	// int/uint/float
	registerIntTypes()
	registerUintTypes()
	registerFloatTypes()

	// string/date/datetime/timestamp
	registerStringTypes()
	registerTimeTypes()

	// enum + common payload types
	registerEnumTypes()
	registerPassThroughTypes()
	registerDBTypes()
}

func registerIntTypes() {
	basic.RegisterType("int", Type{
		Name:  "int",
		Alias: []string{"integer", "int32", "int64", "bigint"},
		Valid: func(value Any, config Var) bool {
			_, ok := builtinToInt64(value)
			return ok
		},
		Value: func(value Any, config Var) Any {
			v, _ := builtinToInt64(value)
			return int(v)
		},
	})
	basic.RegisterType("[int]", Type{
		Name:  "[int]",
		Alias: []string{"array_int", "array_integer", "array_int64", "ints"},
		Valid: func(value Any, config Var) bool {
			switch v := value.(type) {
			case []int:
				return true
			case []Any:
				for _, one := range v {
					if _, ok := builtinToInt64(one); !ok {
						return false
					}
				}
				return true
			default:
				_, ok := builtinToInt64(value)
				return ok
			}
		},
		Value: func(value Any, config Var) Any {
			switch v := value.(type) {
			case []int:
				return v
			case []Any:
				out := make([]int, 0, len(v))
				for _, one := range v {
					n, _ := builtinToInt64(one)
					out = append(out, int(n))
				}
				return out
			default:
				n, _ := builtinToInt64(value)
				return []int{int(n)}
			}
		},
	})
}

func registerUintTypes() {
	basic.RegisterType("uint", Type{
		Name:  "uint",
		Alias: []string{"uint32", "uint64"},
		Valid: func(value Any, config Var) bool {
			n, ok := builtinToInt64(value)
			return ok && n >= 0
		},
		Value: func(value Any, config Var) Any {
			n, _ := builtinToInt64(value)
			if n < 0 {
				return uint(0)
			}
			return uint(n)
		},
	})
	basic.RegisterType("[uint]", Type{
		Name:  "[uint]",
		Alias: []string{"array_uint", "array_uint64", "uints", "units"},
		Valid: func(value Any, config Var) bool {
			switch v := value.(type) {
			case []uint:
				return true
			case []Any:
				for _, one := range v {
					n, ok := builtinToInt64(one)
					if !ok || n < 0 {
						return false
					}
				}
				return true
			default:
				n, ok := builtinToInt64(value)
				return ok && n >= 0
			}
		},
		Value: func(value Any, config Var) Any {
			switch v := value.(type) {
			case []uint:
				return v
			case []Any:
				out := make([]uint, 0, len(v))
				for _, one := range v {
					n, _ := builtinToInt64(one)
					if n < 0 {
						n = 0
					}
					out = append(out, uint(n))
				}
				return out
			default:
				n, _ := builtinToInt64(value)
				if n < 0 {
					n = 0
				}
				return []uint{uint(n)}
			}
		},
	})
}

func registerFloatTypes() {
	basic.RegisterType("float", Type{
		Name:  "float",
		Alias: []string{"number", "double", "decimal"},
		Valid: func(value Any, config Var) bool {
			_, ok := builtinToFloat64(value)
			return ok
		},
		Value: func(value Any, config Var) Any {
			n, _ := builtinToFloat64(value)
			return n
		},
	})
	basic.RegisterType("[float]", Type{
		Name:  "[float]",
		Alias: []string{"array_float", "array_number", "array_double", "floats"},
		Valid: func(value Any, config Var) bool {
			switch v := value.(type) {
			case []float64, []float32:
				return true
			case []Any:
				for _, one := range v {
					if _, ok := builtinToFloat64(one); !ok {
						return false
					}
				}
				return true
			default:
				_, ok := builtinToFloat64(value)
				return ok
			}
		},
		Value: func(value Any, config Var) Any {
			switch v := value.(type) {
			case []float64:
				return v
			case []Any:
				out := make([]float64, 0, len(v))
				for _, one := range v {
					n, _ := builtinToFloat64(one)
					out = append(out, n)
				}
				return out
			default:
				n, _ := builtinToFloat64(value)
				return []float64{n}
			}
		},
	})
}

func registerStringTypes() {
	basic.RegisterType("string", Type{
		Name:  "string",
		Alias: []string{"text"},
		Valid: func(value Any, config Var) bool { return true },
		Value: func(value Any, config Var) Any {
			return builtinToText(value)
		},
	})
	basic.RegisterType("[string]", Type{
		Name:  "[string]",
		Alias: []string{"array_string", "strings", "texts"},
		Valid: func(value Any, config Var) bool { return true },
		Value: func(value Any, config Var) Any {
			return builtinToStringSlice(value)
		},
	})
	basic.RegisterType("[line]", Type{
		Name:  "[line]",
		Valid: func(value Any, config Var) bool { return true },
		Value: func(value Any, config Var) Any {
			lines := strings.Split(builtinToText(value), "\n")
			out := make([]string, 0, len(lines))
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					out = append(out, line)
				}
			}
			return out
		},
	})
}

func registerTimeTypes() {
	basic.RegisterType("date", Type{
		Name: "date",
		Valid: func(value Any, config Var) bool {
			_, ok := builtinParseDate(value)
			return ok
		},
		Value: func(value Any, config Var) Any {
			t, _ := builtinParseDate(value)
			y, m, d := t.Date()
			return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
		},
	})
	basic.RegisterType("[date]", Type{
		Name: "[date]",
		Valid: func(value Any, config Var) bool {
			for _, one := range builtinToSlice(value) {
				if _, ok := builtinParseDate(one); !ok {
					return false
				}
			}
			return true
		},
		Value: func(value Any, config Var) Any {
			out := make([]time.Time, 0)
			for _, one := range builtinToSlice(value) {
				if t, ok := builtinParseDate(one); ok {
					y, m, d := t.Date()
					out = append(out, time.Date(y, m, d, 0, 0, 0, 0, t.Location()))
				}
			}
			return out
		},
	})

	basic.RegisterType("datetime", Type{
		Name: "datetime",
		Valid: func(value Any, config Var) bool {
			_, ok := builtinParseDateTime(value)
			return ok
		},
		Value: func(value Any, config Var) Any {
			t, _ := builtinParseDateTime(value)
			return t
		},
	})
	basic.RegisterType("[datetime]", Type{
		Name: "[datetime]",
		Valid: func(value Any, config Var) bool {
			for _, one := range builtinToSlice(value) {
				if _, ok := builtinParseDateTime(one); !ok {
					return false
				}
			}
			return true
		},
		Value: func(value Any, config Var) Any {
			out := make([]time.Time, 0)
			for _, one := range builtinToSlice(value) {
				if t, ok := builtinParseDateTime(one); ok {
					out = append(out, t)
				}
			}
			return out
		},
	})

	basic.RegisterType("timestamp", Type{
		Name: "timestamp",
		Valid: func(value Any, config Var) bool {
			_, ok := builtinToInt64(value)
			if ok {
				return true
			}
			_, ok = builtinParseDateTime(value)
			return ok
		},
		Value: func(value Any, config Var) Any {
			if n, ok := builtinToInt64(value); ok {
				return n
			}
			t, _ := builtinParseDateTime(value)
			return t.Unix()
		},
	})
	basic.RegisterType("[timestamp]", Type{
		Name: "[timestamp]",
		Valid: func(value Any, config Var) bool {
			for _, one := range builtinToSlice(value) {
				if _, ok := builtinToInt64(one); ok {
					continue
				}
				if _, ok := builtinParseDateTime(one); !ok {
					return false
				}
			}
			return true
		},
		Value: func(value Any, config Var) Any {
			out := make([]int64, 0)
			for _, one := range builtinToSlice(value) {
				if n, ok := builtinToInt64(one); ok {
					out = append(out, n)
					continue
				}
				if t, ok := builtinParseDateTime(one); ok {
					out = append(out, t.Unix())
				}
			}
			return out
		},
	})
}

func registerEnumTypes() {
	enumValues := func(cfg Var) map[string]struct{} {
		out := map[string]struct{}{}
		read := func(src Map) {
			for key, val := range src {
				if key != "enum" && key != "values" && key != "items" {
					continue
				}
				for _, one := range builtinToStringSlice(val) {
					out[one] = struct{}{}
				}
			}
		}
		if cfg.Options != nil {
			read(cfg.Options)
		}
		if cfg.Setting != nil {
			read(cfg.Setting)
		}
		return out
	}

	basic.RegisterType("enum", Type{
		Name: "enum",
		Valid: func(value Any, config Var) bool {
			allowed := enumValues(config)
			if len(allowed) == 0 {
				return true
			}
			_, ok := allowed[builtinToText(value)]
			return ok
		},
		Value: func(value Any, config Var) Any {
			return builtinToText(value)
		},
	})
	basic.RegisterType("[enum]", Type{
		Name: "[enum]",
		Valid: func(value Any, config Var) bool {
			allowed := enumValues(config)
			if len(allowed) == 0 {
				return true
			}
			for _, one := range builtinToStringSlice(value) {
				if _, ok := allowed[one]; !ok {
					return false
				}
			}
			return true
		},
		Value: func(value Any, config Var) Any {
			return builtinToStringSlice(value)
		},
	})
}

func registerPassThroughTypes() {
	for _, name := range []string{"file", "image", "audio", "video"} {
		typeName := name
		basic.RegisterType(typeName, Type{
			Name: typeName,
			Valid: func(value Any, config Var) bool {
				return value != nil
			},
			Value: func(value Any, config Var) Any {
				return value
			},
		})
		basic.RegisterType("["+typeName+"]", Type{
			Name: "[" + typeName + "]",
			Valid: func(value Any, config Var) bool {
				return value != nil
			},
			Value: func(value Any, config Var) Any {
				return builtinToSlice(value)
			},
		})
	}

	basic.RegisterType("json", Type{
		Name:  "json",
		Alias: []string{"jsonb"},
		Valid: func(value Any, config Var) bool {
			switch value.(type) {
			case Map, []Map, []Any:
				return true
			case string, []byte:
				return true
			default:
				return value != nil
			}
		},
		Value: func(value Any, config Var) Any {
			switch v := value.(type) {
			case Map, []Map, []Any:
				return v
			case string:
				var out Any
				if err := json.Unmarshal([]byte(v), &out); err == nil {
					return out
				}
			case []byte:
				var out Any
				if err := json.Unmarshal(v, &out); err == nil {
					return out
				}
			}
			return value
		},
	})
	basic.RegisterType("[json]", Type{
		Name:  "[json]",
		Alias: []string{"array_json", "jsons", "jsonbs"},
		Valid: func(value Any, config Var) bool {
			return value != nil
		},
		Value: func(value Any, config Var) Any {
			switch v := value.(type) {
			case []Any:
				return v
			case []Map:
				out := make([]Any, 0, len(v))
				for _, one := range v {
					out = append(out, one)
				}
				return out
			default:
				return []Any{v}
			}
		},
	})
}

func registerDBTypes() {
	basic.RegisterType("uuid", Type{
		Name: "uuid",
		Valid: func(value Any, config Var) bool {
			return builtinIsUUID(builtinToText(value))
		},
		Value: func(value Any, config Var) Any {
			return strings.ToLower(strings.TrimSpace(builtinToText(value)))
		},
	})
	basic.RegisterType("[uuid]", Type{
		Name: "[uuid]",
		Valid: func(value Any, config Var) bool {
			for _, one := range builtinToSlice(value) {
				if !builtinIsUUID(builtinToText(one)) {
					return false
				}
			}
			return true
		},
		Value: func(value Any, config Var) Any {
			out := make([]string, 0)
			for _, one := range builtinToSlice(value) {
				out = append(out, strings.ToLower(strings.TrimSpace(builtinToText(one))))
			}
			return out
		},
	})

	basic.RegisterType("inet", Type{
		Name: "inet",
		Valid: func(value Any, config Var) bool {
			return net.ParseIP(strings.TrimSpace(builtinToText(value))) != nil
		},
		Value: func(value Any, config Var) Any {
			return strings.TrimSpace(builtinToText(value))
		},
	})
	basic.RegisterType("[inet]", Type{
		Name: "[inet]",
		Valid: func(value Any, config Var) bool {
			for _, one := range builtinToSlice(value) {
				if net.ParseIP(strings.TrimSpace(builtinToText(one))) == nil {
					return false
				}
			}
			return true
		},
		Value: func(value Any, config Var) Any {
			out := make([]string, 0)
			for _, one := range builtinToSlice(value) {
				out = append(out, strings.TrimSpace(builtinToText(one)))
			}
			return out
		},
	})

	basic.RegisterType("cidr", Type{
		Name: "cidr",
		Valid: func(value Any, config Var) bool {
			_, _, err := net.ParseCIDR(strings.TrimSpace(builtinToText(value)))
			return err == nil
		},
		Value: func(value Any, config Var) Any {
			return strings.TrimSpace(builtinToText(value))
		},
	})
	basic.RegisterType("[cidr]", Type{
		Name: "[cidr]",
		Valid: func(value Any, config Var) bool {
			for _, one := range builtinToSlice(value) {
				_, _, err := net.ParseCIDR(strings.TrimSpace(builtinToText(one)))
				if err != nil {
					return false
				}
			}
			return true
		},
		Value: func(value Any, config Var) Any {
			out := make([]string, 0)
			for _, one := range builtinToSlice(value) {
				out = append(out, strings.TrimSpace(builtinToText(one)))
			}
			return out
		},
	})

	basic.RegisterType("macaddr", Type{
		Name: "macaddr",
		Valid: func(value Any, config Var) bool {
			_, err := net.ParseMAC(strings.TrimSpace(builtinToText(value)))
			return err == nil
		},
		Value: func(value Any, config Var) Any {
			text := strings.TrimSpace(builtinToText(value))
			if mac, err := net.ParseMAC(text); err == nil {
				return strings.ToLower(mac.String())
			}
			return strings.ToLower(text)
		},
	})
	basic.RegisterType("[macaddr]", Type{
		Name: "[macaddr]",
		Valid: func(value Any, config Var) bool {
			for _, one := range builtinToSlice(value) {
				_, err := net.ParseMAC(strings.TrimSpace(builtinToText(one)))
				if err != nil {
					return false
				}
			}
			return true
		},
		Value: func(value Any, config Var) Any {
			out := make([]string, 0)
			for _, one := range builtinToSlice(value) {
				text := strings.TrimSpace(builtinToText(one))
				if mac, err := net.ParseMAC(text); err == nil {
					out = append(out, strings.ToLower(mac.String()))
				} else {
					out = append(out, strings.ToLower(text))
				}
			}
			return out
		},
	})

	basic.RegisterType("decimal128", Type{
		Name: "decimal128",
		Valid: func(value Any, config Var) bool {
			_, ok := builtinParseDecimal(value)
			return ok
		},
		Value: func(value Any, config Var) Any {
			if d, ok := builtinParseDecimal(value); ok {
				return d
			}
			return builtinToText(value)
		},
	})
	basic.RegisterType("[decimal128]", Type{
		Name: "[decimal128]",
		Valid: func(value Any, config Var) bool {
			for _, one := range builtinToSlice(value) {
				if _, ok := builtinParseDecimal(one); !ok {
					return false
				}
			}
			return true
		},
		Value: func(value Any, config Var) Any {
			out := make([]string, 0)
			for _, one := range builtinToSlice(value) {
				if d, ok := builtinParseDecimal(one); ok {
					out = append(out, d)
				}
			}
			return out
		},
	})
}

func builtinToText(v Any) string {
	switch vv := v.(type) {
	case string:
		return vv
	case []byte:
		return string(vv)
	default:
		return fmt.Sprintf("%v", vv)
	}
}

func builtinIsUUID(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	if len(v) != 36 {
		return false
	}
	for i, c := range v {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
				return false
			}
		}
	}
	return true
}

func builtinParseDecimal(v Any) (string, bool) {
	text := strings.TrimSpace(builtinToText(v))
	if text == "" {
		return "", false
	}
	r := new(big.Rat)
	if _, ok := r.SetString(text); !ok {
		return "", false
	}
	return r.FloatString(18), true
}

func builtinToBool(v Any) (bool, bool) {
	switch vv := v.(type) {
	case bool:
		return vv, true
	case string:
		s := strings.ToLower(strings.TrimSpace(vv))
		switch s {
		case "1", "true", "yes", "y", "on", "t":
			return true, true
		case "0", "false", "no", "n", "off", "f":
			return false, true
		}
	case int, int8, int16, int32, int64:
		n, _ := builtinToInt64(v)
		return n != 0, true
	case uint, uint8, uint16, uint32, uint64:
		n, _ := builtinToInt64(v)
		return n != 0, true
	case float32, float64:
		f, _ := builtinToFloat64(v)
		return f != 0, true
	}
	return false, false
}

func builtinToInt64(v Any) (int64, bool) {
	switch vv := v.(type) {
	case int:
		return int64(vv), true
	case int8:
		return int64(vv), true
	case int16:
		return int64(vv), true
	case int32:
		return int64(vv), true
	case int64:
		return vv, true
	case uint:
		return int64(vv), true
	case uint8:
		return int64(vv), true
	case uint16:
		return int64(vv), true
	case uint32:
		return int64(vv), true
	case uint64:
		if vv > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(vv), true
	case float32:
		return int64(vv), true
	case float64:
		return int64(vv), true
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(vv), 10, 64)
		return n, err == nil
	}
	return 0, false
}

func builtinToFloat64(v Any) (float64, bool) {
	switch vv := v.(type) {
	case float64:
		return vv, true
	case float32:
		return float64(vv), true
	case int, int8, int16, int32, int64:
		n, ok := builtinToInt64(v)
		return float64(n), ok
	case uint, uint8, uint16, uint32, uint64:
		n, ok := builtinToInt64(v)
		return float64(n), ok
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(vv), 64)
		return f, err == nil
	}
	return 0, false
}

func builtinToSlice(v Any) []Any {
	switch vv := v.(type) {
	case []Any:
		return vv
	case []string:
		out := make([]Any, 0, len(vv))
		for _, one := range vv {
			out = append(out, one)
		}
		return out
	case []int:
		out := make([]Any, 0, len(vv))
		for _, one := range vv {
			out = append(out, one)
		}
		return out
	case []int64:
		out := make([]Any, 0, len(vv))
		for _, one := range vv {
			out = append(out, one)
		}
		return out
	case []Map:
		out := make([]Any, 0, len(vv))
		for _, one := range vv {
			out = append(out, one)
		}
		return out
	default:
		if vv == nil {
			return []Any{}
		}
		return []Any{vv}
	}
}

func builtinToStringSlice(v Any) []string {
	switch vv := v.(type) {
	case []string:
		return vv
	case string:
		s := strings.TrimSpace(vv)
		if s == "" {
			return []string{}
		}
		if (strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")) || (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) {
			s = s[1 : len(s)-1]
		}
		parts := strings.FieldsFunc(s, func(r rune) bool {
			return r == ',' || r == ';' || r == '\n' || r == '\t'
		})
		out := make([]string, 0, len(parts))
		for _, one := range parts {
			one = strings.TrimSpace(one)
			if one != "" {
				out = append(out, one)
			}
		}
		return out
	case []Any:
		out := make([]string, 0, len(vv))
		for _, one := range vv {
			out = append(out, builtinToText(one))
		}
		return out
	default:
		return []string{builtinToText(v)}
	}
}

func builtinParseDate(v Any) (time.Time, bool) {
	if vv, ok := v.(time.Time); ok {
		return vv, true
	}
	if n, ok := builtinToInt64(v); ok {
		if n > 1000000000000 {
			n = n / 1000
		}
		return time.Unix(n, 0), true
	}
	s := strings.TrimSpace(builtinToText(v))
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{"20060102", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func builtinParseDateTime(v Any) (time.Time, bool) {
	if vv, ok := v.(time.Time); ok {
		return vv, true
	}
	if n, ok := builtinToInt64(v); ok {
		if n > 1000000000000 {
			n = n / 1000
		}
		return time.Unix(n, 0), true
	}

	s := strings.TrimSpace(builtinToText(v))
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05.000",
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02T15:04:05.000000Z07:00",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
