# Rebuild the example WASM guest (requires TinyGo + binaryen/wasm-opt).
# https://tinygo.org/getting-started/install/

TINYGO ?= tinygo
OUT ?= hook.wasm

.PHONY: build
build:
	$(TINYGO) build -o $(OUT) -scheduler=none -target=wasip1 -buildmode=c-shared .

.PHONY: clean
clean:
	rm -f $(OUT)
