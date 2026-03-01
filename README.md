# infrago

`infrago` 是框架核心运行时（module: `github.com/infrago/infra`）。

## 包定位

- 类型：核心主包（运行时）
- 作用：统一管理模块生命周期、配置加载、模块装配与调用入口

## 主要功能

- 生命周期：`Load -> Setup -> Open -> Start -> Stop -> Close`
- 统一启动：`infra.Run()`
- 统一调用：`infra.Invoke()`
- 模块挂载：模块通过 `infra.Mount(module)` 接入

## 最小可运行示例

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
