package infra

import (
	"time"

	. "github.com/infrago/base"
)

type TokenSignRequest struct {
	Auth    bool
	Payload Map
	Expires time.Duration
	NewID   bool
	Role    string
	TokenID string
}

type TokenSession struct {
	Token   string
	TokenID string
	Role    string
	Auth    bool
	Payload Map
	Begin   int64
	Expires int64
}
