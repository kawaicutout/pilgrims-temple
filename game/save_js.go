//go:build js && wasm

package game

import (
	"fmt"
	"syscall/js"
)

// Web slots live in localStorage.
const saveSlot = "pilgrims_save"
const scoreSlot = "pilgrims_scores"

// storageStore persists slots in localStorage.
type storageStore struct{}

func defaultStore() slotStore { return storageStore{} }

func openStorage() (js.Value, error) {
	v := js.Global().Get("localStorage")
	if v.IsNull() || v.IsUndefined() {
		return js.Value{}, fmt.Errorf("localStorage unavailable")
	}
	return v, nil
}

func (storageStore) read(slot string) ([]byte, error) {
	ls, err := openStorage()
	if err != nil {
		return nil, err
	}
	v := ls.Call("getItem", slot)
	if v.IsNull() || v.IsUndefined() || v.String() == "" {
		return nil, nil
	}
	return []byte(v.String()), nil
}

func (storageStore) write(slot string, data []byte) error {
	ls, err := openStorage()
	if err != nil {
		return err
	}
	ls.Call("setItem", slot, string(data))
	return nil
}

func (storageStore) delete(slot string) error {
	ls, err := openStorage()
	if err != nil {
		return err
	}
	ls.Call("removeItem", slot)
	return nil
}

func (storageStore) exists(slot string) bool {
	ls, err := openStorage()
	if err != nil {
		return false
	}
	v := ls.Call("getItem", slot)
	return !v.IsNull() && !v.IsUndefined() && v.String() != ""
}
