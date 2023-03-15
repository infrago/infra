package infra

import (
	"io/ioutil"
	"os"
	"sync"
	"time"

	. "github.com/infrago/base"
)

type (
	Meta struct {
		name    string
		payload Map

		attempts int
		final    bool

		language string
		timezone int
		token    string

		mutex     sync.RWMutex
		result    Res
		tempfiles []string

		verify *Token
	}
	Metadata struct {
		Name     string `json:"n,omitempty"`
		Payload  Map    `json:"p,omitempty"`
		Attempts int    `json:"r,omitempty"`
		Final    bool   `json:"f,omitempty"`
		Language string `json:"l,omitempty"`
		Timezone int    `json:"z,omitempty"`
		Token    string `json:"t,omitempty"`
	}
	Echo struct {
		Code  int    `json:"c,omitempty"`
		Text  string `json:"s,omitempty"`
		Args  Vars   `json:"a,omitempty"`
		Value []byte `json:"v,omitempty"`
		Type  string `json:"t,omitempty"`
		Data  Map    `json:"d,omitempty"`
	}
)

// 最终的清理工作
func (meta *Meta) close() {
	for _, file := range meta.tempfiles {
		os.Remove(file)
	}
}

func (meta *Meta) Metadata(datas ...Metadata) Metadata {
	if len(datas) > 0 {
		data := datas[0]
		meta.name = data.Name
		meta.payload = data.Payload
		meta.attempts = data.Attempts
		meta.final = data.Final
		meta.language = data.Language
		meta.timezone = data.Timezone
		meta.token = data.Token

		if data.Token != "" {
			meta.Verify(data.Token)
		}
	}

	return Metadata{
		meta.name, meta.payload, meta.attempts, meta.final, meta.language, meta.timezone, meta.token,
	}
}

// Language 设置的时候，做一下langs的匹配
func (meta *Meta) Language(langs ...string) string {
	if len(langs) > 0 {
		meta.language = langs[0]
	}
	if meta.language == "" {
		return DEFAULT
	}
	return meta.language
}

// Timezone 获取当前时区
func (meta *Meta) Timezone(zones ...*time.Location) *time.Location {
	if len(zones) > 0 {
		_, offset := time.Now().In(zones[0]).Zone()
		meta.timezone = offset
	}
	if meta.timezone == 0 {
		return time.Local
	}
	return time.FixedZone("", meta.timezone)
}

// Attempts 当前是第几次尝试 for queue
func (meta *Meta) Attempts() int {
	return meta.attempts
}

// Retries 当前是第几次重试 for queue
func (meta *Meta) Retries() int {
	return meta.attempts - 1
}

// Final 是否最终尝试 for queue
func (meta *Meta) Final() bool {
	return meta.final
}

// Token 令牌
func (meta *Meta) Token(tokens ...string) string {
	if len(tokens) > 0 {
		meta.token = tokens[0]
	}
	return meta.token
}

// 返回最后的错误信息
// 获取操作结果
func (meta *Meta) Result(res ...Res) Res {
	if len(res) > 0 {
		err := res[0]
		meta.result = err
		return err
	} else {
		if meta.result == nil {
			//nil 也要默认是成功
			//因为 Res = nil的时候，直接 res.Fail() 会报错
			return OK
		}
		err := meta.result
		meta.result = nil
		return err
	}
}

// 获取langString
func (meta *Meta) String(key string, args ...Any) string {
	return infraBasic.String(meta.Language(), key, args...)
}

//----------------------- 签名系统 end ---------------------------------

// ------- 服务调用 -----------------
func (meta *Meta) Invoke(name string, values ...Any) Map {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvv, res := infraEngine.Invoke(meta, name, value)

	meta.result = res

	return vvv
}

func (meta *Meta) Invokes(name string, values ...Any) []Map {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvs, res := infraEngine.Invokes(meta, name, value)

	meta.result = res
	return vvs
}
func (meta *Meta) Invoked(name string, values ...Any) bool {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	vvv, res := infraEngine.Invoked(meta, name, value)
	meta.result = res
	return vvv
}
func (meta *Meta) Invoking(name string, offset, limit int64, values ...Any) (int64, []Map) {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	count, items, res := infraEngine.Invoking(meta, name, offset, limit, value)
	meta.result = res
	return count, items
}

// 集群后，此方法data不好定义，
// 使用gob编码内容后，就不再需要定义data了
func (meta *Meta) Invoker(name string, values ...Any) (Map, []Map) {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	item, items, res := infraEngine.Invoker(meta, name, value)
	meta.result = res
	return item, items
}

func (meta *Meta) Invokee(name string, values ...Any) float64 {
	var value Map
	if len(values) > 0 {
		if vv, ok := values[0].(Map); ok {
			value = vv
		}
	}
	count, res := infraEngine.Invokee(meta, name, value)
	meta.result = res
	return count
}

func (meta *Meta) Library(name string, settings ...Map) *Lib {
	return infraEngine.Library(meta, name, settings...)
}

//------- 服务调用 end-----------------

//待处理

// 生成临时文件
func (meta *Meta) TempFile(patterns ...string) (*os.File, error) {
	meta.mutex.Lock()
	defer meta.mutex.Unlock()

	if meta.tempfiles == nil {
		meta.tempfiles = make([]string, 0)
	}

	file, err := tempFile(patterns...)
	meta.tempfiles = append(meta.tempfiles, file.Name())

	return file, err
}
func (meta *Meta) TempDir(patterns ...string) (string, error) {
	meta.mutex.Lock()
	defer meta.mutex.Unlock()

	if meta.tempfiles == nil {
		meta.tempfiles = make([]string, 0)
	}

	name, err := tempDir(patterns...)
	if err == nil {
		meta.tempfiles = append(meta.tempfiles, name)
	}

	return name, err
}

//token相关

// Id 是token的ID，类似与 sessionId
func (meta *Meta) Id() string {
	if meta.verify != nil {
		// return meta.verify.Header.Id
		return meta.verify.Header.I
	}
	return ""
}

// Tokenized 是否有合法的token
func (meta *Meta) Signed(roles ...string) bool {
	k := ""
	if len(roles) > 0 {
		k = roles[0]
	}
	if meta.verify != nil {
		// if meta.verify.Header.Role == k {
		if meta.verify.Header.R == k {
			return true
		}

	}
	return false
}
func (meta *Meta) Unsigned(roles ...string) bool {
	return false == meta.Signed(roles...)
}

// 是否通过验证
func (meta *Meta) Authed(roles ...string) bool {
	k := ""
	if len(roles) > 0 {
		k = roles[0]
	}
	if meta.verify != nil {
		// if meta.verify.Header.Role == k {
		if meta.verify.Header.R == k {
			// return meta.verify.Header.Auth
			return meta.verify.Header.A
		}

	}
	return false
}
func (meta *Meta) Unauthed(roles ...string) bool {
	return false == meta.Authed(roles...)
}

// Payload Token携带的负载
func (meta *Meta) Payload() Map {
	if meta.verify != nil {
		return meta.verify.Payload
		// return meta.verify.Payload
	}
	return Map{}
}

// Expires Token携带的过期时间
func (meta *Meta) Expires() int64 {
	if meta.verify != nil {
		return meta.verify.Header.E
		// return meta.verify.Header.End
	}
	return -1
}

// Sign 生成签名
// 此方法会覆盖当前上下文的签名
// 要批量生成，请使用infra.Sign
func (meta *Meta) sign(auth bool, payload Map, expires time.Duration, newId bool, roles ...string) string {
	verify := &Token{Payload: payload}
	if newId || meta.Id() == "" {
		verify.Header.I = infraCodec.Generate()
		// verify.Header.Id = infraCodec.Generate()
	} else {
		verify.Header.I = meta.Id()
		// verify.Header.Id = meta.Id()
	}

	if len(roles) > 0 {
		verify.Header.R = roles[0]
		// verify.Header.Role = roles[0]
	}

	// verify.Header.Auth = auth
	verify.Header.A = auth

	if expires > 0 {
		now := time.Now()
		// verify.Header.End = now.Add(ends[0]).Unix()
		verify.Header.E = now.Add(expires).Unix()
	}

	token, err := infraToken.Sign(verify)
	if err != nil {
		meta.Result(errorResult(err))
		return ""
	}

	//这里生成，就替换上下文里的了
	meta.token = token
	meta.verify = verify

	return token
}

// Sign 不会生成新ID
func (meta *Meta) Sign(auth bool, payload Map, expires time.Duration, roles ...string) string {
	return meta.sign(auth, payload, expires, false, roles...)
}

// NewSign 会生成新的ID
func (meta *Meta) NewSign(auth bool, payload Map, expires time.Duration, roles ...string) string {
	return meta.sign(auth, payload, expires, true, roles...)
}

// Verify 验证签名
func (meta *Meta) Verify(token string) error {
	verify, err := infraToken.Verify(token)
	if verify != nil {
		meta.token = token
		meta.verify = verify
	}
	return err
}

//------------------- Process 方法 --------------------

// func (process *Meta) Base(bases ...string) DataBase {
// 	return process.dataBase(bases...)
// }

// CloseMeta 所有携带Meta的Context，必须在执行完成后
// 调用 CloseMeta 来给meta做收尾的工作，主要是删除临时文件，关闭连接之类的
func CloseMeta(meta *Meta) {
	meta.close()
}

func tempFile(patterns ...string) (*os.File, error) {
	pattern := ""
	if len(patterns) > 0 {
		pattern = patterns[0]
	}

	dir := os.TempDir()
	// if infra.config.TempDir != "" {
	// 	dir = infra.config.TempDir
	// }

	return ioutil.TempFile(dir, pattern)
}

func tempDir(patterns ...string) (string, error) {
	pattern := ""
	if len(patterns) > 0 {
		pattern = patterns[0]
	}

	dir := os.TempDir()
	// if mFile.config.TempDir != "" {
	// 	dir = mFile.config.TempDir
	// }

	return ioutil.TempDir(dir, pattern)
}
