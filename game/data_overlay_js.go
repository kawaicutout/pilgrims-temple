//go:build js

package game

import "syscall/js"

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
