package infra

import (
	"fmt"
	"strings"

	. "github.com/infrago/base"
)

var (
	OK       = Result(0, "ok", "成功")
	Fail     = Result(1, "fail", "失败")
	Retry    = Result(2, "retry", "重试").Retry()
	Invalid  = Result(3, "invalid", "无效请求或数据")
	Denied   = Result(4, "denied", "拒绝访问")
	Unsigned = Result(5, "unsigned", "无权访问")
	Unauthed = Result(6, "unauthed", "无权访问")

	varEmpty = Result(7, "varempty", "%s不可为空")
	varError = Result(8, "varerror", "%s无效")
)

type (
	result struct {
		// code 状态码
		// 0 表示成功，其它表示失败
		code int
		// status 对应的状态
		status string
		//携带的参数
		args []Any
		//重试标记
		//专为队列模块
		retry bool
	}
)

// OK 表示Res是否成功
func (res *result) OK() bool {
	if res == nil {
		return true
	}
	return res.code == 0
}

// Fail 表示Res是否失败
func (res *result) Fail() bool {
	return !res.OK()
}

// Code 返回Res的状态码
func (res *result) Code() int {
	return res.code
}

// Status 返回Res的信息
func (res *result) Status() string {
	return res.status
}

// Args 返回Res携带的参数
func (res *result) Args() []Any {
	return res.args
}

// With 使用当前Res加上参数生成一个新的Res并返回
// 因为result都是预先定义好的，所以如果直接修改args，会修改本来已经定义好的result
func (res *result) With(args ...Any) Res {
	if len(args) > 0 {
		return &result{res.code, res.status, args, res.retry}
	}
	return res
}

func (res *result) Retry(flags ...bool) Res {
	retry := true
	if len(flags) > 0 {
		retry = flags[0]
	}
	return &result{res.code, res.status, res.args, retry}
}

func (res *result) Retriable() bool {
	if res == nil {
		return false
	}
	return res.retry
}

// Error
func (res *result) Error() string {
	if res == nil {
		return ""
	}

	text := String(DEFAULT, res.status)

	if res.args != nil && len(res.args) > 0 {
		ccc := strings.Count(text, "%") - strings.Count(text, "%%")
		if ccc == len(res.args) {
			return fmt.Sprintf(text, res.args...)
		}
	}
	return text
}

// String returns the same message as Error, so fmt output is explicit.
func (res *result) String() string {
	return res.Error()
}

func newResult(code int, text string, args ...Any) Res {
	return &result{code, text, args, false}
}
func codeResult(code int, args ...Any) Res {
	return &result{code, "", args, false}
}
func textResult(text string, args ...Any) Res {
	return &result{-1, text, args, false}
}
func errorResult(err error) Res {
	return &result{-1, err.Error(), []Any{}, false}
}

// ErrorResult exposes error-to-Res conversion.
func ErrorResult(err error) Res {
	return errorResult(err)
}

// RetryResult marks one result as retryable and optionally overrides message args.
func RetryResult(res Res, args ...Any) Res {
	if res == nil {
		res = Fail
	}
	if len(args) > 0 {
		res = res.With(args...)
	}
	return res.Retry()
}

// IsRetry reports whether one result is marked as retryable.
func IsRetry(res Res) bool {
	if res == nil {
		return false
	}
	if res.Retriable() {
		return true
	}
	// compatibility with external Res implementations without retry flag support.
	if res == Retry {
		return true
	}
	return res.Status() == Retry.Status()
}

// Result 定义一个result，并自动注册status
// status 表示状态key
// text 表示状态对应的默认文案
func Result(code int, status string, text string) Res {
	//自动注册状态和字串
	basic.RegisterStatus(status, Status(code))
	basic.RegisterStrings(DEFAULT, Strings{status: text})

	// result只携带status，而不携带string
	// 具体的string需要配置context拿到lang之后生成
	// 而实现多语言的状态反馈
	return newResult(code, status)
}

// func NewResult(code int, text string, args ...Any) Res {
// 	return &result{code, text, args}
// }

//-----------------------------------------------

type (
	resultGroup struct {
		name string
		base int
	}
)

//------------ resultGroup ----------------

func (this *resultGroup) Name() string {
	return this.name
}
func (this *resultGroup) Result(ok bool, status string, text string) Res {
	code := 0
	if ok == false {
		code = this.base
		this.base++
	}

	if this.name != "" && !strings.HasPrefix(status, this.name+".") {
		status = this.name + "." + status
	}
	return Result(code, status, text)
}

func ResultGroup(name string, bases ...int) *resultGroup {

	base := 1000
	if len(bases) > 0 {
		base = bases[0]
	}
	return &resultGroup{name, base}
}
