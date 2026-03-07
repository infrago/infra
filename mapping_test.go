package infra

import (
	"testing"

	. "github.com/infrago/base"
)

func TestMappingIgnoresEmptyChildrenOnScalarFields(t *testing.T) {
	out := Map{}
	res := Mapping(Vars{
		"identity": Var{Children: Vars{
			"staff": Var{Children: Vars{
				"id":       Var{Children: Vars{}},
				"avatar":   Var{Children: Vars{}},
				"name":     Var{Children: Vars{}},
				"powerful": Var{Children: Vars{}},
				"title":    Var{Children: Vars{}},
			}},
		}},
	}, Map{
		"identity": Map{
			"staff": Map{
				"id":       1,
				"avatar":   nil,
				"name":     "管理员",
				"powerful": true,
				"title":    nil,
			},
		},
	}, out, false, false)
	if res == nil || res.Fail() {
		t.Fatalf("expected mapping to succeed, got %#v", res)
	}

	staff := out["identity"].(Map)["staff"].(Map)
	if id, ok := staff["id"].(int); !ok || id != 1 {
		t.Fatalf("expected id=1, got %T %#v", staff["id"], staff["id"])
	}
	if name, ok := staff["name"].(string); !ok || name != "管理员" {
		t.Fatalf("expected name to stay string, got %T %#v", staff["name"], staff["name"])
	}
	if powerful, ok := staff["powerful"].(bool); !ok || !powerful {
		t.Fatalf("expected powerful=true, got %T %#v", staff["powerful"], staff["powerful"])
	}
	if avatar, ok := staff["avatar"]; !ok || avatar != nil {
		t.Fatalf("expected avatar=nil, got %T %#v", staff["avatar"], staff["avatar"])
	}
	if title, ok := staff["title"]; !ok || title != nil {
		t.Fatalf("expected title=nil, got %T %#v", staff["title"], staff["title"])
	}
}
