package infra

import (
	"encoding/base64"
	"errors"
	"fmt"
	"hash/fnv"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/infrago/base"
)

var (
	codec = &codecModule{
		config: codecConfig{
			Text:   "01234AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz56789-_/.",
			Digit:  "abcdefghijkmnpqrstuvwxyz123456789ACDEFGHJKLMNPQRSTUVWXYZ",
			Salt:   INFRAGO,
			Length: 7,

			Start:    time.Date(2023, 4, 1, 0, 0, 0, 0, time.Local),
			Timebits: 42, Nodebits: 7, Stepbits: 14,
		},
		codecs: make(map[string]Codec, 0),
	}

	errInvalidCodec     = errors.New("Invalid codec.")
	errInvalidCodecData = errors.New("Invalid codec data.")
)

var (
	// ErrInvalidCodec is returned when codec name is unknown.
	ErrInvalidCodec = errInvalidCodec
	// ErrInvalidCodecData is returned when codec data is invalid.
	ErrInvalidCodecData = errInvalidCodecData
)

const (
	JSON   = "json"
	XML    = "xml"
	GOB    = "gob"
	TOML   = "toml"
	DIGIT  = "digit"
	DIGITS = "digits"
	TEXT   = "text"
	TEXTS  = "texts"
)

type (
	codecConfig struct {
		Text   string
		Digit  string
		Salt   string
		Length int

		Start    time.Time
		Timebits uint
		Nodebits uint
		Stepbits uint
	}

	Codec struct {
		Name   string
		Text   string
		Alias  []string
		Encode EncodeFunc
		Decode DecodeFunc
	}
	Codecs     map[string]Codec
	EncodeFunc func(v Any) (Any, error)
	DecodeFunc func(d Any, v Any) (Any, error)

	codecModule struct {
		mutex  sync.Mutex
		config codecConfig
		codecs map[string]Codec
		fastid *fastID
	}
)

// Register
func (module *codecModule) Register(name string, value Any) {
	switch val := value.(type) {
	case Codec:
		module.RegisterCodec(name, val)
	case Codecs:
		module.RegisterCodecs(val)
	}
}

// Config loads codec config.
func (module *codecModule) Config(global Map) {
	cfg, ok := global["codec"].(Map)
	if !ok {
		return
	}
	if text, ok := cfg["text"].(string); ok && text != "" {
		module.config.Text = text
	}
	if digit, ok := cfg["digit"].(string); ok && digit != "" {
		module.config.Digit = digit
	}
	if salt, ok := cfg["salt"].(string); ok {
		module.config.Salt = salt
	}
	if length, ok := cfg["length"].(int); ok {
		module.config.Length = length
	}
	if length, ok := cfg["length"].(int64); ok {
		module.config.Length = int(length)
	}
	if vv, ok := cfg["start"].(time.Time); ok {
		module.config.Start = vv
	}
	if vv, ok := cfg["start"].(int64); ok {
		module.config.Start = time.Unix(vv, 0)
	}
	if vv, ok := cfg["timebits"].(int); ok {
		module.config.Timebits = uint(vv)
	}
	if vv, ok := cfg["timebits"].(int64); ok {
		module.config.Timebits = uint(vv)
	}
	if vv, ok := cfg["nodebits"].(int); ok {
		module.config.Nodebits = uint(vv)
	}
	if vv, ok := cfg["nodebits"].(int64); ok {
		module.config.Nodebits = uint(vv)
	}
	if vv, ok := cfg["stepbits"].(int); ok {
		module.config.Stepbits = uint(vv)
	}
	if vv, ok := cfg["stepbits"].(int64); ok {
		module.config.Stepbits = uint(vv)
	}
}

func (module *codecModule) Setup() {
	module.fastid = newFastID(module.config.Timebits, module.config.Nodebits, module.config.Stepbits, module.config.Start.Unix())
}
func (module *codecModule) Open()  {}
func (module *codecModule) Start() {}
func (module *codecModule) Stop()  {}
func (module *codecModule) Close() {}

// RegisterCodec registers one codec.
func (module *codecModule) RegisterCodec(name string, config Codec) {
	module.mutex.Lock()
	defer module.mutex.Unlock()

	alias := make([]string, 0)
	if name != "" {
		alias = append(alias, name)
	}
	if config.Alias != nil {
		alias = append(alias, config.Alias...)
	}

	for _, key := range alias {
		if Override() {
			module.codecs[key] = config
		} else {
			if _, ok := module.codecs[key]; !ok {
				module.codecs[key] = config
			}
		}
	}
}

// RegisterCodecs registers codecs in batch.
func (module *codecModule) RegisterCodecs(codecs Codecs) {
	for name, cfg := range codecs {
		module.RegisterCodec(name, cfg)
	}
}

// ListCodecs returns all registered codecs.
func (module *codecModule) ListCodecs() map[string]Codec {
	codecs := map[string]Codec{}
	for k, v := range module.codecs {
		codecs[k] = v
	}
	return codecs
}

// Sequence returns snowflake id.
func (module *codecModule) Sequence() int64 {
	if module.fastid == nil {
		module.Setup()
	}
	return module.fastid.NextID()
}

// Generate returns hex id (simple, fast).
func (module *codecModule) Generate(prefixs ...string) string {
	id := module.Sequence()
	return strconv.FormatInt(id, 16)
}

// Encode
func (module *codecModule) Encode(codecName string, v Any) (Any, error) {
	codecName = strings.ToLower(codecName)
	if ccc, ok := module.codecs[codecName]; ok {
		return ccc.Encode(v)
	}
	return nil, errInvalidCodec
}

// Decode
func (module *codecModule) Decode(codecName string, d Any, v Any) (Any, error) {
	codecName = strings.ToLower(codecName)
	if ccc, ok := module.codecs[codecName]; ok {
		return ccc.Decode(d, v)
	}
	return nil, errInvalidCodec
}

// Marshal
func (module *codecModule) Marshal(codecName string, v Any) ([]byte, error) {
	dat, err := module.Encode(codecName, v)
	if err != nil {
		return nil, err
	}
	if bts, ok := dat.([]byte); ok {
		return bts, nil
	}
	return nil, errInvalidCodecData
}

// Unmarshal
func (module *codecModule) Unmarshal(codecName string, d []byte, v Any) error {
	_, err := module.Decode(codecName, d, v)
	return err
}

// Encrypt returns string.
func (module *codecModule) Encrypt(codecName string, v Any) (string, error) {
	dat, err := module.Encode(codecName, v)
	if err != nil {
		return "", err
	}
	switch vv := dat.(type) {
	case string:
		return vv, nil
	case []byte:
		return string(vv), nil
	default:
		return fmt.Sprintf("%v", vv), nil
	}
}

// Decrypt
func (module *codecModule) Decrypt(codecName string, v Any) (Any, error) {
	return module.Decode(codecName, v, nil)
}

// wrappers
func Encode(name string, v Any) (Any, error)             { return codec.Encode(name, v) }
func Decode(name string, data Any, obj Any) (Any, error) { return codec.Decode(name, data, obj) }
func Marshal(name string, obj Any) ([]byte, error)       { return codec.Marshal(name, obj) }
func Unmarshal(name string, data []byte, obj Any) error  { return codec.Unmarshal(name, data, obj) }
func Encrypt(name string, obj Any) (string, error)       { return codec.Encrypt(name, obj) }
func Decrypt(name string, obj Any) (Any, error)          { return codec.Decrypt(name, obj) }

// RegisterCodec registers one codec implementation.
func RegisterCodec(name string, config Codec) {
	codec.RegisterCodec(name, config)
}

// RegisterCodecs registers codec implementations in batch.
func RegisterCodecs(config Codecs) {
	codec.RegisterCodecs(config)
}

// CodecTextAlphabet returns configured text alphabet.
func CodecTextAlphabet() string {
	return codec.config.Text
}

// CodecDigitAlphabet returns configured digit alphabet.
func CodecDigitAlphabet() string {
	return codec.config.Digit
}

// CodecSalt returns configured codec salt.
func CodecSalt() string {
	return codec.config.Salt
}

// CodecLength returns configured encoded length.
func CodecLength() int {
	return codec.config.Length
}

// EncodeInt64 encodes one int64 using current codec config.
func EncodeInt64(v int64) (string, error) {
	return encodeInt64(v, codec.config.Digit, codec.config.Salt, codec.config.Length)
}

// DecodeInt64 decodes one int64 using current codec config.
func DecodeInt64(v string) (int64, error) {
	return decodeInt64(v, codec.config.Digit, codec.config.Salt)
}

// EncodeInt64Slice encodes int64 slice using current codec config.
func EncodeInt64Slice(v []int64) (string, error) {
	return encodeInt64Slice(v, codec.config.Digit, codec.config.Salt, codec.config.Length)
}

// DecodeInt64Slice decodes int64 slice using current codec config.
func DecodeInt64Slice(v string) ([]int64, error) {
	return decodeInt64Slice(v, codec.config.Digit, codec.config.Salt)
}

// EncodeTextBytes encodes bytes using current codec text config.
func EncodeTextBytes(v []byte) (string, error) {
	return encodeBytes(v, codec.config.Text, codec.config.Salt)
}

// DecodeTextBytes decodes bytes using current codec text config.
func DecodeTextBytes(v string) ([]byte, error) {
	return decodeBytes(v, codec.config.Text, codec.config.Salt)
}

func Sequence() int64                   { return codec.Sequence() }
func Generate(prefixs ...string) string { return codec.Generate(prefixs...) }

func normalizeAlphabet(alphabet string) ([]rune, error) {
	if alphabet == "" {
		return nil, errInvalidCodecData
	}
	seen := map[rune]bool{}
	out := make([]rune, 0, len(alphabet))
	for _, r := range []rune(alphabet) {
		if seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	if len(out) < 2 {
		return nil, errInvalidCodecData
	}
	return out, nil
}

func rotateAlphabet(alpha []rune, salt string) []rune {
	if len(alpha) == 0 {
		return alpha
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(salt))
	off := int(h.Sum32()) % len(alpha)
	return append(alpha[off:], alpha[:off]...)
}

func pickSeparator(alpha []rune) (string, error) {
	cands := []string{"-", "_", ".", "~", "|", ":", ","}
	set := map[rune]bool{}
	for _, r := range alpha {
		set[r] = true
	}
	for _, c := range cands {
		if !set[rune(c[0])] {
			return c, nil
		}
	}
	return "", errInvalidCodecData
}

func encodeInt64(n int64, alphabet, salt string, minLen int) (string, error) {
	if n < 0 {
		return "", errInvalidCodecData
	}
	alpha, err := normalizeAlphabet(alphabet)
	if err != nil {
		return "", err
	}
	alpha = rotateAlphabet(alpha, salt)
	if n == 0 {
		return string(alpha[0]), nil
	}
	base := int64(len(alpha))
	var out []rune
	for n > 0 {
		r := n % base
		out = append(out, alpha[r])
		n = n / base
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	if minLen > 0 && len(out) < minLen {
		pad := alpha[0]
		for len(out) < minLen {
			out = append([]rune{pad}, out...)
		}
	}
	return string(out), nil
}

func decodeInt64(s, alphabet, salt string) (int64, error) {
	alpha, err := normalizeAlphabet(alphabet)
	if err != nil {
		return 0, err
	}
	alpha = rotateAlphabet(alpha, salt)
	index := map[rune]int64{}
	for i, r := range alpha {
		index[r] = int64(i)
	}
	var n int64
	for _, r := range []rune(s) {
		v, ok := index[r]
		if !ok {
			return 0, errInvalidCodecData
		}
		n = n*int64(len(alpha)) + v
	}
	return n, nil
}

func encodeInt64Slice(ns []int64, alphabet, salt string, minLen int) (string, error) {
	sep, err := pickSeparator([]rune(alphabet))
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(ns))
	for _, n := range ns {
		enc, err := encodeInt64(n, alphabet, salt, minLen)
		if err != nil {
			return "", err
		}
		parts = append(parts, enc)
	}
	return strings.Join(parts, sep), nil
}

func decodeInt64Slice(s, alphabet, salt string) ([]int64, error) {
	sep, err := pickSeparator([]rune(alphabet))
	if err != nil {
		return nil, err
	}
	if s == "" {
		return []int64{}, nil
	}
	parts := strings.Split(s, sep)
	out := make([]int64, 0, len(parts))
	for _, p := range parts {
		val, err := decodeInt64(p, alphabet, salt)
		if err != nil {
			return nil, err
		}
		out = append(out, val)
	}
	return out, nil
}

func encodeBytes(data []byte, alphabet, salt string) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	alpha, err := normalizeAlphabet(alphabet)
	if err != nil {
		return "", err
	}
	alpha = rotateAlphabet(alpha, salt)
	base := big.NewInt(int64(len(alpha)))
	n := new(big.Int).SetBytes(data)
	if n.Sign() == 0 {
		return string(alpha[0]), nil
	}
	var out []rune
	r := new(big.Int)
	for n.Sign() > 0 {
		n, r = new(big.Int).DivMod(n, base, r)
		out = append(out, alpha[r.Int64()])
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out), nil
}

func decodeBytes(s, alphabet, salt string) ([]byte, error) {
	if s == "" {
		return []byte{}, nil
	}
	alpha, err := normalizeAlphabet(alphabet)
	if err != nil {
		return nil, err
	}
	alpha = rotateAlphabet(alpha, salt)
	index := map[rune]int64{}
	for i, r := range alpha {
		index[r] = int64(i)
	}
	base := big.NewInt(int64(len(alpha)))
	n := big.NewInt(0)
	for _, r := range []rune(s) {
		v, ok := index[r]
		if !ok {
			return nil, errInvalidCodecData
		}
		n.Mul(n, base)
		n.Add(n, big.NewInt(v))
	}
	return n.Bytes(), nil
}

// fallback: base64 urlsafe when alphabet is invalid (not used by default)
func encodeBase64URL(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeBase64URL(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// fast id (snowflake-ish)
type fastID struct {
	timeStart int64
	timeBits  uint
	stepBits  uint
	nodeBits  uint
	timeMask  int64
	stepMask  int64
	nodeID    int64
	lastID    int64
}

func newFastID(timeBits, nodeBits, stepBits uint, timeStart int64) *fastID {
	machineID := int64(0)
	timeMask := ^(int64(-1) << timeBits)
	stepMask := ^(int64(-1) << stepBits)
	nodeMask := ^(int64(-1) << nodeBits)
	return &fastID{
		timeStart: timeStart,
		timeBits:  timeBits,
		stepBits:  stepBits,
		nodeBits:  nodeBits,
		timeMask:  timeMask,
		stepMask:  stepMask,
		nodeID:    machineID & nodeMask,
		lastID:    0,
	}
}

func (f *fastID) currentTimestamp() int64 {
	return (time.Now().UnixNano() - f.timeStart) >> 20 & f.timeMask
}

func (f *fastID) NextID() int64 {
	for {
		localLast := f.lastID
		seq := f.sequence(localLast)
		lastTime := f.time(localLast)
		now := f.currentTimestamp()
		if now > lastTime {
			seq = 0
		} else if seq >= f.stepMask {
			time.Sleep(time.Duration(0xFFFFF - (time.Now().UnixNano() & 0xFFFFF)))
			continue
		} else {
			seq++
		}
		newID := now<<(f.nodeBits+f.stepBits) + seq<<f.nodeBits + f.nodeID
		if newID > localLast {
			f.lastID = newID
			return newID
		}
		time.Sleep(time.Duration(20))
	}
}

func (f *fastID) sequence(id int64) int64 { return (id >> f.nodeBits) & f.stepMask }
func (f *fastID) time(id int64) int64     { return id >> (f.nodeBits + f.stepBits) }
