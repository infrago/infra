package infra

import (
	"testing"

	. "github.com/infrago/base"
)

func TestInvokeListTopLevel(t *testing.T) {
	originalCore := core
	core = &coreModule{
		entries: map[string]coreEntry{
			"demo.list": {
				Action: func(*Context) Map {
					return Map{
						"summary": "ok",
						"items":   []Map{{"id": 1}, {"id": 2}},
					}
				},
			},
		},
	}
	defer func() {
		core = originalCore
	}()

	data, items := InvokeList("demo.list")
	if data["summary"] != "ok" {
		t.Fatalf("expected summary=ok, got %#v", data["summary"])
	}
	if _, ok := data["items"]; ok {
		t.Fatalf("expected returned item to not include items")
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestInvokeListMeta(t *testing.T) {
	originalCore := core
	core = &coreModule{
		entries: map[string]coreEntry{
			"demo.list": {
				Action: func(*Context) []Map {
					return []Map{{"id": 1}, {"id": 2}, {"id": 3}}
				},
			},
		},
	}
	defer func() {
		core = originalCore
	}()

	meta := NewMeta()
	data, items := meta.InvokeList("demo.list")
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if len(data) != 0 {
		t.Fatalf("expected returned item to be empty map, got %#v", data)
	}
	if res := meta.Result(); res == nil || res.Fail() {
		t.Fatalf("expected meta result to be ok")
	}
}

func TestInvokeItemsFromAnySlice(t *testing.T) {
	items := invokeItems(Map{
		"items": []Any{
			Map{"id": 1},
			map[string]Any{"id": 2},
			"skip",
		},
	})
	if len(items) != 2 {
		t.Fatalf("expected 2 map items, got %d", len(items))
	}
}

func TestInvokeItemsMissing(t *testing.T) {
	if got := invokeItems(nil); len(got) != 0 {
		t.Fatalf("expected empty items for nil data, got %d", len(got))
	}
	if got := invokeItems(Map{}); len(got) != 0 {
		t.Fatalf("expected empty items for missing key, got %d", len(got))
	}
}

func TestInvokeListSupportsActionReturnMapAndItems(t *testing.T) {
	originalCore := core
	core = &coreModule{
		entries: map[string]coreEntry{
			"demo.combo": {
				Action: func(*Context) (Map, []Map) {
					return Map{"summary": "ok"}, []Map{{"id": 1}, {"id": 2}}
				},
			},
		},
	}
	defer func() {
		core = originalCore
	}()

	data, items := InvokeList("demo.combo")
	if data["summary"] != "ok" {
		t.Fatalf("expected summary=ok, got %#v", data["summary"])
	}
	if _, ok := data["items"]; ok {
		t.Fatalf("expected returned item to not include items")
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestInvokeListSupportsActionReturnMapItemsRes(t *testing.T) {
	originalCore := core
	core = &coreModule{
		entries: map[string]coreEntry{
			"demo.combo.res": {
				Action: func(*Context) (Map, []Map, Res) {
					return Map{"summary": "ok"}, []Map{{"id": 1}}, OK
				},
			},
		},
	}
	defer func() {
		core = originalCore
	}()

	meta := NewMeta()
	data, items := meta.InvokeList("demo.combo.res")
	if data["summary"] != "ok" {
		t.Fatalf("expected summary=ok, got %#v", data["summary"])
	}
	if _, ok := data["items"]; ok {
		t.Fatalf("expected returned item to not include items")
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if res := meta.Result(); res == nil || res.Fail() {
		t.Fatalf("expected meta result to be ok")
	}
}

func TestInvokesCallsActionOnceAndReturnsItems(t *testing.T) {
	originalCore := core
	called := 0
	core = &coreModule{
		entries: map[string]coreEntry{
			"demo.invokes": {
				Action: func(ctx *Context) []Map {
					called++
					return []Map{{"id": ctx.Value["id"]}}
				},
			},
		},
	}
	defer func() {
		core = originalCore
	}()

	items, res := Invokes("demo.invokes", Map{"id": 1}, Map{"id": 2})
	if res == nil || res.Fail() {
		t.Fatalf("expected invokes result to be ok")
	}
	if called != 1 {
		t.Fatalf("expected action to be called once, got %d", called)
	}
	if len(items) != 1 || items[0]["id"] != 1 {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestInvokingPagesReturnedItems(t *testing.T) {
	originalCore := core
	core = &coreModule{
		entries: map[string]coreEntry{
			"demo.page": {
				Action: func(*Context) []Map {
					return []Map{{"id": 1}, {"id": 2}, {"id": 3}}
				},
			},
		},
	}
	defer func() {
		core = originalCore
	}()

	total, items := Invoking("demo.page", 1, 1)
	if total != 3 {
		t.Fatalf("expected total=3, got %d", total)
	}
	if len(items) != 1 || items[0]["id"] != 2 {
		t.Fatalf("unexpected paged items: %#v", items)
	}
}

func TestInvokeListSupportsActionReturnItemsAndRes(t *testing.T) {
	originalCore := core
	core = &coreModule{
		entries: map[string]coreEntry{
			"demo.items.res": {
				Action: func(*Context) ([]Map, Res) {
					return []Map{{"id": 9}}, OK
				},
			},
		},
	}
	defer func() {
		core = originalCore
	}()

	items, res := Invokes("demo.items.res")
	if res == nil || res.Fail() {
		t.Fatalf("expected invokes result to be ok")
	}
	if len(items) != 1 || items[0]["id"] != 9 {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestInvokingSupportsActionReturnTotalAndItems(t *testing.T) {
	originalCore := core
	core = &coreModule{
		entries: map[string]coreEntry{
			"demo.total.items": {
				Action: func(*Context) (int64, []Map) {
					return 100, []Map{{"id": 11}, {"id": 12}}
				},
			},
		},
	}
	defer func() {
		core = originalCore
	}()

	total, items := Invoking("demo.total.items", 1, 1)
	if total != 100 {
		t.Fatalf("expected total=100, got %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("expected items length=2, got %d", len(items))
	}
}

func TestInvokingSupportsActionReturnTotalItemsAndRes(t *testing.T) {
	originalCore := core
	core = &coreModule{
		entries: map[string]coreEntry{
			"demo.total.items.res": {
				Action: func(*Context) (int64, []Map, Res) {
					return 7, []Map{{"id": 1}}, OK
				},
			},
		},
	}
	defer func() {
		core = originalCore
	}()

	meta := NewMeta()
	total, items := meta.Invoking("demo.total.items.res", 0, 1)
	if total != 7 {
		t.Fatalf("expected total=7, got %d", total)
	}
	if len(items) != 1 || items[0]["id"] != 1 {
		t.Fatalf("unexpected items: %#v", items)
	}
	if res := meta.Result(); res == nil || res.Fail() {
		t.Fatalf("expected meta result to be ok")
	}
}
