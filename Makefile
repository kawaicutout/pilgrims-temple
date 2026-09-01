GO      ?= go
BIN      = pilgrims-temple
WASM     = web/main.wasm
WASM_BR  = web/main.wasm.br
WASM_EXEC_JS = web/wasm_exec.js

LDFLAGS  = -s -w

.PHONY: all run terminal wasm wasm-br web zip clean vet test

all: terminal wasm

# ---- terminal ----
terminal:
	$(GO) build -ldflags="$(LDFLAGS)" -o $(BIN) ./cmd/terminal

run: terminal
	./$(BIN)

# ---- wasm ----
wasm: $(WASM) $(WASM_EXEC_JS)

$(WASM): cmd/wasm/*.go game/*.go game/data/*.json
	GOOS=js GOARCH=wasm $(GO) build -ldflags="$(LDFLAGS)" -o $(WASM) ./cmd/wasm
	@if command -v wasm-opt >/dev/null 2>&1; then \
		echo "wasm-opt -O3 $(WASM)"; \
		wasm-opt --enable-bulk-memory --enable-nontrapping-float-to-int --enable-sign-ext --enable-mutable-globals -O3 $(WASM) -o $(WASM) || echo "wasm-opt failed — keeping unoptimized $(WASM)"; \
	fi
	@ls -lh $(WASM)

$(WASM_EXEC_JS):
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" $(WASM_EXEC_JS)

wasm-br: wasm $(WASM_BR)

$(WASM_BR): $(WASM)
	@if command -v brotli >/dev/null 2>&1; then \
		brotli -q 11 -f $(WASM) -o $(WASM_BR) && ls -lh $(WASM) $(WASM_BR); \
	else \
		echo "brotli not found — skipping $(WASM_BR)"; \
	fi

# Both variants in one go (user requested: produce both web and web-brotli)
web: wasm wasm-br

# itch.io upload: zip of web/ with brotli wasm only (DESIGN 13.3)
zip: wasm-br
	cd web && zip -r ../pilgrims-temple-web.zip index.html tokens.css wasm_exec.js main.wasm.br
	@ls -lh pilgrims-temple-web.zip

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

clean:
	rm -f $(BIN) $(WASM) $(WASM_BR) pilgrims-temple-web.zip
	rm -f $(WASM_EXEC_JS)
