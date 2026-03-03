package infra

import (
	"context"
	"strings"
	"sync"
	"time"

	. "github.com/infrago/base"
)

type (
	Meta struct {
		mutex sync.RWMutex
		ctx   context.Context

		traceId      string
		spanId       string
		parentSpanId string
		traceKind    string
		traceEntry   string

		language string
		timezone int
		token    string

		result Res

		payload    Map
		tokenId    string
		tokenValid bool
		tokenAuth  bool

		spanStack []metaSpanFrame
	}

	Metadata struct {
		TraceId      string `json:"tid,omitempty"`
		SpanId       string `json:"sid,omitempty"`
		ParentSpanId string `json:"psid,omitempty"`
		Language     string `json:"l,omitempty"`
		Timezone     int    `json:"z,omitempty"`
		Token        string `json:"t,omitempty"`
	}

	metaSpanFrame struct {
		prevSpanId   string
		prevParentId string
		prevKind     string
		prevEntry    string
	}
)

func NewMeta() *Meta {
	return &Meta{ctx: context.Background(), spanStack: make([]metaSpanFrame, 0, 8)}
}

func (m *Meta) WithContext(ctx context.Context) *Meta {
	if ctx == nil {
		ctx = context.Background()
	}
	m.ctx = ctx
	return m
}

func (m *Meta) Context() context.Context {
	if m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

func (m *Meta) TraceId(id ...string) string {
	if len(id) > 0 {
		m.traceId = id[0]
	}
	return m.traceId
}

func (m *Meta) SpanId(id ...string) string {
	if len(id) > 0 {
		m.spanId = id[0]
	}
	return m.spanId
}

func (m *Meta) ParentSpanId(id ...string) string {
	if len(id) > 0 {
		m.parentSpanId = id[0]
	}
	return m.parentSpanId
}

func (m *Meta) TraceKind(kind ...string) string {
	if len(kind) > 0 {
		m.traceKind = kind[0]
	}
	return m.traceKind
}

func (m *Meta) TraceEntry(entry ...string) string {
	if len(entry) > 0 {
		m.traceEntry = entry[0]
	}
	return m.traceEntry
}

func (m *Meta) PushSpanFrame(prevSpanId, prevParentId string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.spanStack = append(m.spanStack, metaSpanFrame{
		prevSpanId:   prevSpanId,
		prevParentId: prevParentId,
		prevKind:     m.traceKind,
		prevEntry:    m.traceEntry,
	})
}

func (m *Meta) PopSpanFrame() (string, string, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	size := len(m.spanStack)
	if size == 0 {
		return "", "", false
	}
	last := m.spanStack[size-1]
	m.spanStack = m.spanStack[:size-1]
	m.traceKind = last.prevKind
	m.traceEntry = last.prevEntry
	return last.prevSpanId, last.prevParentId, true
}

// ParseTraceParent parses W3C traceparent: 00-<traceid>-<spanid>-<flags>.
func (m *Meta) ParseTraceParent(traceparent string) bool {
	traceparent = strings.TrimSpace(traceparent)
	if traceparent == "" {
		return false
	}
	parts := strings.Split(traceparent, "-")
	if len(parts) != 4 {
		return false
	}
	traceId := strings.TrimSpace(parts[1])
	spanId := strings.TrimSpace(parts[2])
	if len(traceId) != 32 || len(spanId) != 16 || !isHexString(traceId) || !isHexString(spanId) {
		return false
	}
	m.TraceId(traceId)
	m.ParentSpanId(spanId)
	return true
}

// TraceParent builds W3C traceparent using current trace/span ids.
func (m *Meta) TraceParent() string {
	traceId := normalizeHexID(m.TraceId(), 32)
	spanId := normalizeHexID(m.SpanId(), 16)
	return "00-" + traceId + "-" + spanId + "-01"
}

func normalizeHexID(raw string, size int) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	raw = strings.TrimPrefix(raw, "0x")
	filtered := make([]rune, 0, len(raw))
	for _, r := range raw {
		switch {
		case r >= '0' && r <= '9':
			filtered = append(filtered, r)
		case r >= 'a' && r <= 'f':
			filtered = append(filtered, r)
		}
	}
	raw = string(filtered)
	if len(raw) > size {
		return raw[len(raw)-size:]
	}
	if len(raw) < size {
		return strings.Repeat("0", size-len(raw)) + raw
	}
	if raw == "" {
		return strings.Repeat("0", size)
	}
	return raw
}

func isHexString(v string) bool {
	for _, r := range v {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func (m *Meta) Language(v ...string) string {
	if len(v) > 0 {
		m.language = v[0]
	}
	return m.language
}

// String returns localized string by language.
func (m *Meta) String(key string, args ...Any) string {
	return String(m.Language(), key, args...)
}
func (m *Meta) Timezone(zones ...*time.Location) *time.Location {
	if len(zones) > 0 {
		_, offset := time.Now().In(zones[0]).Zone()
		m.timezone = offset
	}
	if m.timezone == 0 {
		return time.Local
	}
	return time.FixedZone("", m.timezone)
}

func (m *Meta) Token(v ...string) string {
	if len(v) > 0 {
		m.token = v[0]
		m.clearTokenState()
	}
	return m.token
}

// Verify validates token signature and payload.
func (m *Meta) Verify(token string) error {
	if token == "" {
		m.token = ""
		m.clearTokenState()
		return nil
	}

	m.token = token
	m.clearTokenState()

	session, err := hook.VerifyToken(token)
	if err != nil {
		return err
	}

	m.tokenId = session.TokenID
	m.payload = session.Payload
	m.tokenAuth = session.Auth
	m.tokenValid = true
	return nil
}

// Sign issues token with current token id.
// expires is optional duration, begin defaults to current time.
func (m *Meta) Sign(auth bool, payload Map, expires ...time.Duration) string {
	return m.SignAt(auth, payload, time.Now(), expires...)
}

// SignAt issues token with current token id and custom begin time.
// expires is optional duration.
func (m *Meta) SignAt(auth bool, payload Map, begin time.Time, expires ...time.Duration) string {
	beginUnix, expireUnix := tokenTimeWindow(begin, expires...)
	tokenID := m.tokenId
	if tokenID == "" {
		tokenID = Generate()
	}
	req := Token{
		Auth:    auth,
		Payload: payload,
		Begin:   beginUnix,
		Expires: expireUnix,
		NewID:   false,
		TokenID: tokenID,
	}
	token, err := hook.SignToken(req)
	if err != nil {
		m.Result(errorResult(err))
		return ""
	}
	if req.Payload == nil {
		req.Payload = Map{}
	}
	m.token = token
	m.tokenId = req.TokenID
	m.payload = req.Payload
	m.tokenAuth = req.Auth
	m.tokenValid = true
	return token
}

// NewSign issues token with a new token id.
// expires is optional duration, begin defaults to current time.
func (m *Meta) NewSign(auth bool, payload Map, expires ...time.Duration) string {
	return m.NewSignAt(auth, payload, time.Now(), expires...)
}

// NewSignAt issues token with a new token id and custom begin time.
// expires is optional duration.
func (m *Meta) NewSignAt(auth bool, payload Map, begin time.Time, expires ...time.Duration) string {
	beginUnix, expireUnix := tokenTimeWindow(begin, expires...)
	req := Token{
		Auth:    auth,
		Payload: payload,
		Begin:   beginUnix,
		Expires: expireUnix,
		NewID:   true,
		TokenID: Generate(),
	}
	token, err := hook.SignToken(req)
	if err != nil {
		m.Result(errorResult(err))
		return ""
	}
	if req.Payload == nil {
		req.Payload = Map{}
	}
	m.token = token
	m.tokenId = req.TokenID
	m.payload = req.Payload
	m.tokenAuth = req.Auth
	m.tokenValid = true
	return token
}

func tokenTimeWindow(begin time.Time, expires ...time.Duration) (int64, int64) {
	if begin.IsZero() {
		begin = time.Now()
	}
	beginUnix := begin.Unix()
	expireUnix := int64(0)
	if len(expires) > 0 && expires[0] > 0 {
		expireUnix = beginUnix + int64(expires[0].Seconds())
	}
	return beginUnix, expireUnix
}

// RevokeToken revokes one raw token.
func (m *Meta) RevokeToken(token string, expires ...int64) error {
	exp := int64(0)
	if len(expires) > 0 {
		exp = expires[0]
	}
	return hook.RevokeToken(token, exp)
}

// RevokeTokenID revokes one token id.
func (m *Meta) RevokeTokenID(tokenID string, expires ...int64) error {
	exp := int64(0)
	if len(expires) > 0 {
		exp = expires[0]
	}
	return hook.RevokeTokenID(tokenID, exp)
}

// Signed returns whether token is valid.
func (m *Meta) Signed() bool {
	return m.tokenValid
}

// Unsigned is the negation of Signed.
func (m *Meta) Unsigned() bool {
	return !m.Signed()
}

// Authed returns whether token is valid and auth flag is true.
func (m *Meta) Authed() bool {
	return m.tokenValid && m.tokenAuth
}

// Unauthed is the negation of Authed.
func (m *Meta) Unauthed() bool {
	return !m.Authed()
}

// TokenId returns token id placeholder.
func (m *Meta) TokenId() string {
	return m.tokenId
}

// Payload returns token payload placeholder.
func (m *Meta) Payload() Map {
	if m.payload == nil {
		return Map{}
	}
	return m.payload
}

func (m *Meta) Result(res ...Res) Res {
	if len(res) > 0 {
		m.result = res[0]
		return res[0]
	}

	if m.result == nil {
		return OK
	}

	ret := m.result
	m.result = nil
	return ret
}

func (m *Meta) Metadata(data ...Metadata) Metadata {
	if len(data) > 0 {
		d := data[0]
		m.traceId = d.TraceId
		m.spanId = d.SpanId
		m.parentSpanId = d.ParentSpanId
		m.language = d.Language
		m.timezone = d.Timezone
		m.Token(d.Token)
		if d.Token != "" {
			_ = m.Verify(d.Token)
		}
	}

	return Metadata{
		TraceId:      m.traceId,
		SpanId:       m.spanId,
		ParentSpanId: m.parentSpanId,
		Language:     m.language,
		Timezone:     m.timezone,
		Token:        m.token,
	}
}

// Begin starts a trace span through trace hook.
func (m *Meta) Begin(name string, attrs ...Map) TraceSpan {
	merged := mergeMetaAttrs(attrs...)
	return hook.Begin(m, name, merged)
}

// Trace emits one trace event through trace hook.
func (m *Meta) Trace(name string, attrs ...Map) error {
	merged := mergeMetaAttrs(attrs...)
	status := ""
	if v, ok := merged["status"].(string); ok {
		status = v
		delete(merged, "status")
	}
	return hook.Trace(m, name, status, merged)
}

// Invoke calls another service (local first, then bus).
// It stores the result in meta and returns only the data.
func (m *Meta) Invoke(name string, values ...Map) Map {
	var value Map
	if len(values) > 0 {
		value = values[0]
	}
	data, res := core.Invoke(m, name, value)
	m.result = res
	return data
}

// Invokes executes multiple calls in order and stores last result on meta.
func (m *Meta) Invokes(name string, values ...Map) []Map {
	if len(values) == 0 {
		m.result = OK
		return []Map{}
	}
	results := make([]Map, 0, len(values))
	for _, value := range values {
		data, res := core.Invoke(m, name, value)
		m.result = res
		if res != nil && res.Fail() {
			return results
		}
		results = append(results, data)
	}
	m.result = OK
	return results
}

// Invoking executes a paged subset of calls and stores last result on meta.
func (m *Meta) Invoking(name string, offset, limit int, values ...Map) (int64, []Map) {
	total := int64(len(values))
	start, end := normalizeInvokeWindow(len(values), offset, limit)
	if start >= end {
		m.result = OK
		return total, []Map{}
	}
	results := make([]Map, 0, end-start)
	for _, value := range values[start:end] {
		data, res := core.Invoke(m, name, value)
		m.result = res
		if res != nil && res.Fail() {
			return total, results
		}
		results = append(results, data)
	}
	m.result = OK
	return total, results
}

// InvokeOK executes one call and returns whether result is OK.
func (m *Meta) InvokeOK(name string, values ...Map) bool {
	_ = m.Invoke(name, values...)
	return m.result == nil || m.result.OK()
}

// InvokeFail executes one call and returns whether result is failed.
func (m *Meta) InvokeFail(name string, values ...Map) bool {
	return !m.InvokeOK(name, values...)
}

func mergeMetaAttrs(items ...Map) Map {
	out := Map{}
	for _, item := range items {
		if item == nil {
			continue
		}
		for k, v := range item {
			out[k] = v
		}
	}
	return out
}

func (m *Meta) clearTokenState() {
	m.tokenValid = false
	m.tokenAuth = false
	m.tokenId = ""
	m.payload = nil
}

// Context carries invocation data for method/service.
type Context struct {
	*Meta

	Name    string
	Config  *coreEntry
	Setting Map
	Value   Map
	Args    Map
}
