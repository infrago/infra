package infra

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	. "github.com/infrago/base"
)

var (
	errInvalidToken     = errors.New("invalid token")
	errInvalidTokenSign = errors.New("invalid token sign")
	errInvalidTokenTime = errors.New("invalid token time window")
)

type defaultTokenHook struct {
	mutex          sync.Mutex
	revokedTokens  map[string]int64
	revokedTokenID map[string]int64
}

type defaultTokenHeader struct {
	ID    string `json:"i,omitempty"`
	Begin int64  `json:"b,omitempty"`
	End   int64  `json:"e,omitempty"`
	Auth  bool   `json:"a,omitempty"`
	Role  string `json:"r,omitempty"`
}

func newDefaultTokenHook() *defaultTokenHook {
	return &defaultTokenHook{
		revokedTokens:  make(map[string]int64),
		revokedTokenID: make(map[string]int64),
	}
}

func (h *defaultTokenHook) Sign(_ *Meta, req TokenSignRequest) (TokenSession, error) {
	now := time.Now().Unix()

	tokenID := req.TokenID
	if tokenID == "" || req.NewID {
		tokenID = Generate()
	}

	header := defaultTokenHeader{
		ID:   tokenID,
		Auth: req.Auth,
		Role: req.Role,
	}
	if req.Expires > 0 {
		header.End = now + int64(req.Expires.Seconds())
	}

	payload := req.Payload
	if payload == nil {
		payload = Map{}
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return TokenSession{}, err
	}
	headerText, err := EncodeTextBytes(headerBytes)
	if err != nil {
		return TokenSession{}, err
	}

	payloadBytes, err := Marshal(defaultTokenCodec(), payload)
	if err != nil {
		return TokenSession{}, err
	}
	payloadText, err := EncodeTextBytes(payloadBytes)
	if err != nil {
		return TokenSession{}, err
	}

	unsigned := headerText + "." + payloadText
	signature, err := defaultTokenHMACSign(unsigned, defaultTokenSecret())
	if err != nil {
		return TokenSession{}, err
	}

	token := unsigned + "." + signature
	return TokenSession{
		Token:   token,
		TokenID: tokenID,
		Role:    req.Role,
		Auth:    req.Auth,
		Payload: payload,
		Begin:   header.Begin,
		Expires: header.End,
	}, nil
}

func (h *defaultTokenHook) Verify(_ *Meta, token string) (TokenSession, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return TokenSession{}, errInvalidToken
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return TokenSession{}, errInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]
	if !defaultTokenHMACVerify(unsigned, parts[2], defaultTokenSecret()) {
		return TokenSession{}, errInvalidTokenSign
	}

	headerBytes, err := DecodeTextBytes(parts[0])
	if err != nil {
		return TokenSession{}, err
	}
	payloadBytes, err := DecodeTextBytes(parts[1])
	if err != nil {
		return TokenSession{}, err
	}

	header := defaultTokenHeader{}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return TokenSession{}, err
	}

	now := time.Now().Unix()
	if header.Begin > 0 && now < header.Begin {
		return TokenSession{}, errInvalidTokenTime
	}
	if header.End > 0 && now > header.End {
		return TokenSession{}, errInvalidTokenTime
	}

	if h.isRevokedToken(token, now) || h.isRevokedTokenID(header.ID, now) {
		return TokenSession{}, errInvalidToken
	}

	payload := Map{}
	if err := Unmarshal(defaultTokenCodec(), payloadBytes, &payload); err != nil {
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return TokenSession{}, err
		}
	}
	if payload == nil {
		payload = Map{}
	}

	return TokenSession{
		Token:   token,
		TokenID: header.ID,
		Role:    header.Role,
		Auth:    header.Auth,
		Payload: payload,
		Begin:   header.Begin,
		Expires: header.End,
	}, nil
}

func (h *defaultTokenHook) RevokeToken(_ *Meta, token string, expires int64) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.revokedTokens[token] = expires
	return nil
}

func (h *defaultTokenHook) RevokeTokenID(_ *Meta, tokenID string, expires int64) error {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return nil
	}
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.revokedTokenID[tokenID] = expires
	return nil
}

func (h *defaultTokenHook) isRevokedToken(token string, now int64) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	exp, ok := h.revokedTokens[token]
	if !ok {
		return false
	}
	if exp > 0 && now > exp {
		delete(h.revokedTokens, token)
		return false
	}
	return true
}

func (h *defaultTokenHook) isRevokedTokenID(tokenID string, now int64) bool {
	if tokenID == "" {
		return false
	}
	h.mutex.Lock()
	defer h.mutex.Unlock()
	exp, ok := h.revokedTokenID[tokenID]
	if !ok {
		return false
	}
	if exp > 0 && now > exp {
		delete(h.revokedTokenID, tokenID)
		return false
	}
	return true
}

func defaultTokenSecret() string {
	if env := strings.TrimSpace(os.Getenv("INFRAGO_TOKEN_SECRET")); env != "" {
		return env
	}
	project, _, _ := infrago.runtimeInfo()
	if project != "" {
		return project
	}
	return INFRAGO
}

func defaultTokenCodec() string {
	if v := strings.TrimSpace(defaultTokenSetting("token.codec")); v != "" {
		return v
	}
	return GOB
}

func defaultTokenSetting(key string) string {
	infrago.mutex.RLock()
	defer infrago.mutex.RUnlock()
	if v, ok := infrago.setting[key].(string); ok {
		return v
	}
	return ""
}

func defaultTokenHMACSign(data string, key string) (string, error) {
	if key == "" {
		return "", errors.New("empty token secret")
	}
	hasher := hmac.New(sha1.New, []byte(key))
	if _, err := hasher.Write([]byte(data)); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil)), nil
}

func defaultTokenHMACVerify(data, sign, key string) bool {
	sig, err := base64.URLEncoding.DecodeString(sign)
	if err != nil {
		sig, err = base64.RawURLEncoding.DecodeString(sign)
		if err != nil {
			return false
		}
	}
	hasher := hmac.New(sha1.New, []byte(key))
	if _, err := hasher.Write([]byte(data)); err != nil {
		return false
	}
	return hmac.Equal(sig, hasher.Sum(nil))
}

func formatTokenRole(roles ...string) string {
	if len(roles) == 0 {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", roles[0]))
}
