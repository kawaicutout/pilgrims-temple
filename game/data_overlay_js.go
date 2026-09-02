//go:build js

package game

import (
	"strings"
	"syscall/js"
)

func overlayJSON(name string) ([]byte, bool) {
	ls := js.Global().Get("localStorage")
	if ls.IsNull() || ls.IsUndefined() {
		return nil, false
	}
	v := ls.Call("getItem", "data:"+name)
	if v.IsNull() || v.IsUndefined() {
		return nil, false
	}
	s := v.String()
	if s == "" {
		return nil, false
	}
	return []byte(s), true
}

func HasModifiedData() bool {
	ls := js.Global().Get("localStorage")
	if ls.IsNull() || ls.IsUndefined() {
		return false
	}
	n := ls.Get("length").Int()
	for i := 0; i < n; i++ {
		k := ls.Call("key", i)
		if k.IsNull() || k.IsUndefined() {
			continue
		}
		if strings.HasPrefix(k.String(), "data:") {
			return true
		}
	}
	return false
}
