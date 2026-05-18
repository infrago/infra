package infra

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultTokenHMACSignUsesRawBase64URL(t *testing.T) {
	sig, err := defaultTokenHMACSign("payload", "secret")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if strings.Contains(sig, "=") {
		t.Fatalf("signature should be raw base64url without padding: %q", sig)
	}
	if !defaultTokenHMACVerify("payload", sig, "secret") {
		t.Fatalf("raw signature should verify")
	}
}

func TestMetaRevokeDefaultsToCurrentTokenExpires(t *testing.T) {
	tokenHook := newDefaultTokenHook()
	hook.AttachToken(tokenHook)
	defer hook.AttachToken(newDefaultTokenHook())

	exp := time.Now().Add(time.Hour).Unix()
	meta := NewMeta()
	meta.token = "current-token"
	meta.tokenId = "current-token-id"
	meta.tokenExpires = exp

	if err := meta.RevokeTokenID(meta.tokenId); err != nil {
		t.Fatalf("revoke token id: %v", err)
	}
	if got := tokenHook.revokedTokenID[meta.tokenId]; got != exp {
		t.Fatalf("expected token id revoke expiry %d, got %d", exp, got)
	}

	if err := meta.RevokeToken(meta.token); err != nil {
		t.Fatalf("revoke token: %v", err)
	}
	if got := tokenHook.revokedTokens[meta.token]; got != exp {
		t.Fatalf("expected token revoke expiry %d, got %d", exp, got)
	}
}

func TestMetaRevokeCurrentHelpers(t *testing.T) {
	tokenHook := newDefaultTokenHook()
	hook.AttachToken(tokenHook)
	defer hook.AttachToken(newDefaultTokenHook())

	exp := time.Now().Add(time.Hour).Unix()
	meta := NewMeta()
	meta.token = "current-token"
	meta.tokenId = "current-token-id"
	meta.tokenExpires = exp

	if err := meta.RevokeCurrentTokenID(); err != nil {
		t.Fatalf("revoke current token id: %v", err)
	}
	if got := tokenHook.revokedTokenID[meta.tokenId]; got != exp {
		t.Fatalf("expected current token id revoke expiry %d, got %d", exp, got)
	}

	if err := meta.RevokeCurrentToken(); err != nil {
		t.Fatalf("revoke current token: %v", err)
	}
	if got := tokenHook.revokedTokens[meta.token]; got != exp {
		t.Fatalf("expected current token revoke expiry %d, got %d", exp, got)
	}
}
