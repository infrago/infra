# infrago

`infrago` 主运行时（module: `github.com/infrago/infra`），负责生命周期、配置加载、模块装配和统一调用入口。

## 安装

```bash
go get github.com/infrago/infra@latest
```

## 快速启动

```go
package main

import (
    _ "github.com/infrago/http"
    _ "github.com/infrago/log"
    "github.com/infrago/infra"
)

func main() {
    infra.Run()
}
```

```toml
[infrago]
name = "demo"
profile = "dev"

[http]
port = 8100

[log]
driver = "default"
```

## 生命周期

- `Load` -> `Setup` -> `Open` -> `Start` -> `Stop` -> `Close`

## 公开 API（摘自源码）

- `func AssetFS(fss ...fs.FS) fs.FS`
- `func AssetDir(name string) ([]fs.DirEntry, error)`
- `func AssetFile(name string) ([]byte, error)`
- `func AssetStat(name string) (fs.FileInfo, error)`
- `func Mount(mod Module) Host`
- `func Register(args ...Any)`
- `func RegisterProfile(key string, profile Profile)`
- `func Prepare(profile ...string)`
- `func Ready(profile ...string)`
- `func Run(profile ...string)`
- `func Go(profile ...string)`
- `func Override(args ...bool) bool`
- `func Identity() infragoIdentity`
- `func Node() string`
- `func Invoke(name string, values ...Map) (Map, Res)`
- `func Invokes(name string, values ...Map) ([]Map, Res)`
- `func Invoking(name string, offset, limit int, values ...Map) (int64, []Map)`
- `func InvokeOK(name string, values ...Map) bool`
- `func InvokeFail(name string, values ...Map) bool`
- `func Enqueue(name string, value Map) error`
- `func Broadcast(name string, value Map) error`
- `func Publish(name string, value Map) error`
- `func (m *libraryModule) Register(name string, value Any)`
- `func (m *libraryModule) RegisterLibrary(prefix string, def Library)`
- `func (m *libraryModule) Config(Map) {}`
- `func (m *libraryModule) Setup()     {}`
- `func (m *libraryModule) Open()`
- `func (m *libraryModule) Start() {}`
- `func (m *libraryModule) Stop()  {}`
- `func (m *libraryModule) Close() {}`
- `func (m *Meta) Library(name string, settings ...Map) *libraryInvoker`
- `func (l *libraryInvoker) Invoke(name string, values ...Map) Map`
- `func (l *libraryInvoker) Result() Res`
- `func (m *libraryModule) Load(name string) (Library, bool)`
- `func TraceAttrs(service, kind, entry string, attrs ...Map) Map`
- `func (c *infragoRuntime) Name() string`
- `func (c *infragoRuntime) Project() string`
- `func (c *infragoRuntime) Profile() string`
- `func (c *infragoRuntime) Node() string`
- `func (c *infragoRuntime) Identity() infragoIdentity`

## 常见问题

- 模块未生效：确认已 `import _ "github.com/infrago/<module>"`
- 配置未读取：确认当前目录存在 `config.toml`/`infra.toml`/`infrago.toml`
