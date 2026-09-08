//go:build js

package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"syscall/js"

	"partyrogue/game"
)

// setupShellControls wires data zip upload + reset + minimal JSON editor
// (localStorage overlay "data:"+name). Ported from cmd/wasm; the renderer
// draws the game on canvas, the DOM keeps the data tooling.
func setupShellControls(doc js.Value) {
	if zipInput := doc.Call("getElementById", "dataZip"); !zipInput.IsNull() {
		var onZipChange js.Func
		onZipChange = js.FuncOf(func(this js.Value, args []js.Value) any {
			files := zipInput.Get("files")
			if files.Get("length").Int() == 0 {
				return nil
			}
			file := files.Index(0)
			reader := js.Global().Get("FileReader").New()
			var onLoad js.Func
			onLoad = js.FuncOf(func(this js.Value, args []js.Value) any {
				defer onLoad.Release()
				buf := reader.Get("result")
				u8 := js.Global().Get("Uint8Array").New(buf)
				n := u8.Get("length").Int()
				data := make([]byte, n)
				js.CopyBytesToGo(data, u8)
				r, err := zip.NewReader(bytes.NewReader(data), int64(n))
				shellStatus := doc.Call("getElementById", "shellStatus")
				if err != nil {
					if !shellStatus.IsNull() {
						shellStatus.Set("textContent", "zip error: "+err.Error())
					}
					return nil
				}
				ls := js.Global().Get("localStorage")
				count := 0
				for _, f := range r.File {
					if f.FileInfo().IsDir() {
						continue
					}
					rc, err := f.Open()
					if err != nil {
						continue
					}
					b, err := io.ReadAll(rc)
					rc.Close()
					if err != nil {
						continue
					}
					name := path.Base(f.Name)
					if name == "" || name == "." {
						continue
					}
					ls.Call("setItem", "data:"+name, string(b))
					count++
				}
				if !shellStatus.IsNull() {
					shellStatus.Set("textContent", fmt.Sprintf("Stored %d files — reloading…", count))
				}
				js.Global().Get("location").Call("reload")
				return nil
			})
			reader.Set("onload", onLoad)
			reader.Call("readAsArrayBuffer", file)
			return nil
		})
		zipInput.Call("addEventListener", "change", onZipChange)
	}
	if resetBtn := doc.Call("getElementById", "resetData"); !resetBtn.IsNull() {
		resetBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
			ls := js.Global().Get("localStorage")
			if !ls.IsNull() && !ls.IsUndefined() {
				lsLen := ls.Get("length").Int()
				toRemove := []string{}
				for i := range lsLen {
					k := ls.Call("key", i)
					if k.IsNull() || k.IsUndefined() {
						continue
					}
					ks := k.String()
					if strings.HasPrefix(ks, "data:") {
						toRemove = append(toRemove, ks)
					}
				}
				for _, k := range toRemove {
					ls.Call("removeItem", k)
				}
			}
			shellStatus := doc.Call("getElementById", "shellStatus")
			if !shellStatus.IsNull() {
				shellStatus.Set("textContent", "Reset — reloading defaults…")
			}
			js.Global().Get("location").Call("reload")
			return nil
		}))
	}
	editorEl := doc.Call("getElementById", "editor")
	editSelect := doc.Call("getElementById", "editSelect")
	editArea := doc.Call("getElementById", "editArea")
	editStatus := doc.Call("getElementById", "editStatus")
	editLoad := doc.Call("getElementById", "editLoad")
	editSave := doc.Call("getElementById", "editSave")
	toggleBtn := doc.Call("getElementById", "toggleEditor")
	loadIntoEditor := func() {
		if editSelect.IsNull() || editArea.IsNull() {
			return
		}
		name := editSelect.Get("value").String()
		if name == "" {
			name = "tuning.json"
		}
		b, err := game.RawJSON(name)
		if err != nil {
			if !editStatus.IsNull() {
				editStatus.Set("textContent", "load error: "+err.Error())
			}
			return
		}
		var m json.RawMessage
		if json.Unmarshal(b, &m) == nil {
			var pretty bytes.Buffer
			if json.Indent(&pretty, b, "", "  ") == nil {
				b = pretty.Bytes()
			}
		}
		editArea.Set("value", string(b))
		if !editStatus.IsNull() {
			editStatus.Set("textContent", "Loaded "+name+" ("+strconv.FormatInt(int64(len(b)), 10)+" bytes)")
		}
	}
	if !editLoad.IsNull() {
		editLoad.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
			loadIntoEditor()
			return nil
		}))
	}
	if !editSelect.IsNull() {
		editSelect.Call("addEventListener", "change", js.FuncOf(func(this js.Value, args []js.Value) any {
			loadIntoEditor()
			return nil
		}))
	}
	if !editSave.IsNull() {
		editSave.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
			if editSelect.IsNull() || editArea.IsNull() {
				return nil
			}
			name := editSelect.Get("value").String()
			if name == "" {
				name = "tuning.json"
			}
			txt := editArea.Get("value").String()
			if !json.Valid([]byte(txt)) {
				if !editStatus.IsNull() {
					editStatus.Set("textContent", "Invalid JSON — not saved")
				}
				return nil
			}
			ls := js.Global().Get("localStorage")
			if !ls.IsNull() && !ls.IsUndefined() {
				ls.Call("setItem", "data:"+name, txt)
			}
			if !editStatus.IsNull() {
				editStatus.Set("textContent", "Saved "+name+" to overlay — reload to apply")
			}
			return nil
		}))
	}
	if !toggleBtn.IsNull() && !editorEl.IsNull() {
		toggleBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
			cls := editorEl.Get("classList")
			if cls.Call("contains", "open").Bool() {
				cls.Call("remove", "open")
			} else {
				cls.Call("add", "open")
				loadIntoEditor()
			}
			return nil
		}))
	}
}

// setupShell wires the DOM data tooling; call once at boot on web.
func setupShell() {
	setupShellControls(js.Global().Get("document"))
}

// viewportFit scales the canvas target box: the web game fills 80% of the
// viewport in each dimension (whichever binds), never the full window.
func viewportFit() float64 { return 0.8 }
