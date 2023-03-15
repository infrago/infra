package infra

type (
	env = int
)

const (
	_ env = iota
	developing
	testing
	preview
	production
	//
)

const (
	INFRA   = "infra"
	INFRAGO = "infra.go"

	DEFAULT = "default"

	UTF8   = "utf-8"
	GB2312 = "gb2312"
	GBK    = "gbk"

	_EMPTY = ""
)
