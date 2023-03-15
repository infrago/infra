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
	INFRAGO = "infra.go"
	_EMPTY  = ""
)
