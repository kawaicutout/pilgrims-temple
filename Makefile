GO      ?= go
BIN      = pilgrims-temple
BIN_LINUX   = bin/pilgrims-temple
BIN_WINDOWS = bin/pilgrims-temple.exe
WASM     = web/main.wasm
WASM_BR  = web/main.wasm.br
WASM_EXEC_JS = web/wasm_exec.js

LDFLAGS  = -s -w

.PHONY: all run terminal wasm wasm-br web zip bin clean vet test
all: terminal wasm

run: terminal
	./$(BIN)

# ---- bin (cross-platform) ----
bin: bin-linux bin-windows

bin-linux: $(BIN_LINUX)
bin-windows: $(BIN_WINDOWS)

$(BIN_LINUX): cmd/terminal/*.go game/*.go game/data/*.json
	mkdir -p bin
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_LINUX) ./cmd/terminal
	@ls -lh $(BIN_LINUX)

$(BIN_WINDOWS): cmd/terminal/*.go game/*.go game/data/*.json
	mkdir -p bin
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_WINDOWS) ./cmd/terminal
	@ls -lh $(BIN_WINDOWS)


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

# Deploy web to GitHub Pages (legacy gh-pages branch, no Actions needed)
deploy-pages: web
	@echo "Deploying web/ to gh-pages branch..."
	@git rev-parse --verify gh-pages >/dev/null 2>&1 || git branch gh-pages
	@worktree="$$(mktemp -d)"; \
	git worktree add --detach "$$worktree" gh-pages 2>/dev/null || git worktree add "$$worktree" gh-pages; \
	rm -rf "$$worktree"/*; \
	cp web/index.html web/tokens.css web/wasm_exec.js web/main.wasm web/main.wasm.br "$$worktree"/; \
	cd "$$worktree" && git add index.html tokens.css wasm_exec.js main.wasm main.wasm.br && git commit -m "Deploy web $$(date -u +%Y-%m-%dT%H:%M:%SZ)" && git push origin HEAD:gh-pages --force; \
	git worktree remove --force "$$worktree"; \
	rmdir "$$worktree" 2>/dev/null || true
	@echo "Deployed to https://kawaicutout.github.io/pilgrims-temple/"

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

clean:
	rm -f $(BIN) $(WASM) $(WASM_BR) pilgrims-temple-web.zip
	rm -f $(WASM_EXEC_JS)
	rm -rf bin
