package infra

import (
	"testing"

	. "github.com/infrago/base"
)

func TestArgumentsReturnsMethodArgsCopy(t *testing.T) {
	m := &coreModule{
		entries: map[string]coreEntry{
			"demo.method": {
				Args: Vars{
					"id":   Var{Type: "int"},
					"name": Var{Type: "string"},
				},
			},
		},
	}

	args := m.Arguments("demo.method")
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args["id"].Type != "int" || args["name"].Type != "string" {
		t.Fatalf("unexpected args: %#v", args)
	}

	args["id"] = Var{Type: "float"}
	if m.entries["demo.method"].Args["id"].Type != "int" {
		t.Fatalf("expected source args to stay unchanged")
	}
}

func TestArgumentsSupportsExtendAndDelete(t *testing.T) {
	originalCore := core
	core = &coreModule{
		entries: map[string]coreEntry{
			"demo.service": {
				Args: Vars{
					"id":   Var{Type: "int"},
					"name": Var{Type: "string"},
				},
			},
		},
	}
	defer func() {
		core = originalCore
	}()

	args := Arguments("demo.service", Vars{
		"id":    Var{Type: "float"},
		"name":  Nil,
		"extra": Var{Type: "bool"},
	})

	if _, ok := args["name"]; ok {
		t.Fatalf("expected name arg to be removed")
	}
	if args["id"].Type != "float" {
		t.Fatalf("expected id arg to be overridden, got %#v", args["id"])
	}
	if args["extra"].Type != "bool" {
		t.Fatalf("expected extra arg to be appended, got %#v", args["extra"])
	}
}
