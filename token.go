package infra

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	. "github.com/infrago/base"
	"github.com/infrago/util"
)

var (
	ErrInvalidToken = errors.New("Invalid token.")
)

var (
	infraToken = &tokenModule{
		config: tokenConfig{
			Secret: "",
		},
	}
)

type (
	tokenConfig struct {
		Codec  string
		Secret string
		Expire time.Duration
	}

	// tokenHeader struct {
	// 	Id    string `json:"d,omitempty"`
	// 	Begin int64  `json:"b,omitempty"`
	// 	End   int64  `json:"e,omitempty"`
	// 	Auth  bool   `json:"a,omitempty"`
	// 	Role  bool   `json:"r,omitempty"`
	// }

	//for gob shorter
	tH struct {
		I string `json:"i,omitempty"`
		B int64  `json:"b,omitempty"`
		E int64  `json:"e,omitempty"`
		A bool   `json:"a,omitempty"`
		R string `json:"r,omitempty"`
	}

	Token struct {
		Header  tH  `json:"h,omitempty"`
		Payload Map `json:"p,omitempty"`
	}

	tokenModule struct {
		mutex  sync.Mutex
		config tokenConfig
	}
)

// Register
func (module *tokenModule) Register(name string, value Any) {
	// switch val := value.(type) {
	// // case What:
	// }
}

// Configure
func (this *tokenModule) Configure(global Map) {
	var config Map
	if vv, ok := global["token"].(Map); ok {
		config = vv
	}

	if codec, ok := config["codec"].(string); ok {
		this.config.Codec = codec
	}
	if secret, ok := config["secret"].(string); ok {
		this.config.Secret = secret
	}

	//默认过期时间，单位秒
	if expire, ok := config["expire"].(string); ok {
		dur, err := util.ParseDuration(expire)
		if err == nil {
			this.config.Expire = dur
		}
	}
	if expire, ok := config["expire"].(int); ok {
		this.config.Expire = time.Second * time.Duration(expire)
	}
	if expire, ok := config["expire"].(int64); ok {
		this.config.Expire = time.Second * time.Duration(expire)
	}
	if expire, ok := config["expire"].(float64); ok {
		this.config.Expire = time.Second * time.Duration(expire)
	}
}

func (this *tokenModule) Initialize() {
	if this.config.Codec == "" {
		this.config.Codec = GOB
	}
	if this.config.Secret == "" {
		if infra.config.name != "" {
			this.config.Secret = infra.config.name
		} else {
			this.config.Secret = INFRAGO
		}
	}
}
func (this *tokenModule) Connect() {
}
func (this *tokenModule) Launch() {
}
func (this *tokenModule) Terminate() {
}

//------------------------- 方法 ----------------------------

func (this *tokenModule) Sign(token *Token) (string, error) {
	header, payload := "{}", "{}"

	//header指定类型，所有用json
	if vv, err := infraCodec.MarshalJSON(token.Header); err != nil {
		return "", err
	} else {
		if vvs, err := infraCodec.Encrypt(TEXT, string(vv)); err != nil {
			return "", err
		} else {
			header = vvs
		}
	}

	if vv, err := infraCodec.Marshal(this.config.Codec, token.Payload); err != nil {
		return "", err
	} else {
		if vvs, err := infraCodec.Encrypt(TEXT, vv); err != nil {
			return "", err
		} else {
			payload = vvs
		}
	}

	tokenString := header + "." + payload

	//计算签名
	sign, err := util.HMACSign(tokenString, this.config.Secret)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.%s", tokenString, sign), nil
}

func (this *tokenModule) Verify(str string) (*Token, error) {

	alls := strings.Split(str, ".")
	if len(alls) != 3 {
		return nil, ErrInvalidToken
	}

	header := alls[0]
	payload := alls[1]
	sign := alls[2]

	tokenString := header + "." + payload

	//验证签名
	err := util.HMACVerify(tokenString, sign, this.config.Secret)
	if err != nil {
		return nil, err
	}

	token := &Token{}

	if vv, err := infraCodec.Decrypt(TEXT, header); err != nil {
		return nil, err
	} else {
		var bytes []byte
		if vvs, ok := vv.([]byte); ok {
			bytes = vvs
		} else if vvs, ok := vv.(string); ok {
			bytes = []byte(vvs)
		}

		//header指定类型，所有用JSON
		if err := infraCodec.UnmarshalJSON(bytes, &token.Header); err != nil {
			return nil, err
		}
	}

	if vv, err := infraCodec.Decrypt(TEXT, payload); err != nil {
		return nil, err
	} else {
		var bytes []byte
		if vvs, ok := vv.([]byte); ok {
			bytes = vvs
		} else if vvs, ok := vv.(string); ok {
			bytes = []byte(vvs)
		}

		if err := infraCodec.Unmarshal(this.config.Codec, bytes, &token.Payload); err != nil {
			return nil, err
		}
	}

	now := time.Now()

	//是否校验，并且在有效期以内
	if token.Header.B > 0 && now.Unix() < token.Header.B {
		token.Header.A = false
	}
	if token.Header.E > 0 && now.Unix() > token.Header.E {
		token.Header.A = false
	}
	// if token.Header.Begin > 0 && now.Unix() < token.Header.Begin {
	// 	token.Header.Auth = false
	// }
	// if token.Header.End > 0 && now.Unix() > token.Header.End {
	// 	token.Header.Auth = false
	// }

	return token, nil
}

// -------------  外部方法 -------

// Sign 生成签名
// 可以用在一些批量生成的场景
func Sign(auth bool, payload Map, expires time.Duration, roles ...string) string {
	verify := &Token{Payload: payload}

	// verify.Header.Id = infraCodec.Generate()
	verify.Header.I = infraCodec.Generate()
	// verify.Header.Auth = auth
	verify.Header.A = auth

	if len(roles) > 0 {
		// verify.Header.Role = roles[0]
		verify.Header.R = roles[0]
	}

	if expires > 0 {
		now := time.Now()
		// verify.Header.End = now.Add(ends[0]).Unix()
		verify.Header.E = now.Add(expires).Unix()
	}

	token, err := infraToken.Sign(verify)
	if err != nil {
		return ""
	}
	return token
}

// Verify
func Verify(token string) (*Token, error) {
	return infraToken.Verify(token)
}
