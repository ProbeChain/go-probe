# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gprobe android ios gprobe-cross evm all test clean
.PHONY: gprobe-linux gprobe-linux-386 gprobe-linux-amd64 gprobe-linux-mips64 gprobe-linux-mips64le
.PHONY: gprobe-linux-arm gprobe-linux-arm-5 gprobe-linux-arm-6 gprobe-linux-arm-7 gprobe-linux-arm64
.PHONY: gprobe-darwin gprobe-darwin-386 gprobe-darwin-amd64
.PHONY: gprobe-windows gprobe-windows-386 gprobe-windows-amd64

GOBIN = ./build/bin
GO ?= latest
GORUN = env GO111MODULE=on go run

gprobe:
	$(GORUN) build/ci.go install ./cmd/gprobe
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gprobe\" to launch gprobe."

all:
	$(GORUN) build/ci.go install

android:
	$(GORUN) build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/gprobe.aar\" to use the library."
	@echo "Import \"$(GOBIN)/gprobe-sources.jar\" to add javadocs"
	@echo "For more info see https://stackoverflow.com/questions/20994336/android-studio-how-to-attach-javadoc"

ios:
	$(GORUN) build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Gprobe.framework\" to use the library."

test: all
	$(GORUN) build/ci.go test

lint: ## Run linters.
	$(GORUN) build/ci.go lint

clean:
	env GO111MODULE=on go clean -cache
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go install golang.org/x/tools/cmd/stringer@latest
	env GOBIN= go install github.com/kevinburke/go-bindata/go-bindata@latest
	env GOBIN= go install github.com/fjl/gencodec@latest
	env GOBIN= go install github.com/golang/protobuf/protoc-gen-go@latest
	env GOBIN= go install ./cmd/abigen
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

gprobe-cross: gprobe-linux gprobe-darwin gprobe-windows gprobe-android gprobe-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-*

gprobe-linux: gprobe-linux-386 gprobe-linux-amd64 gprobe-linux-arm gprobe-linux-mips64 gprobe-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-linux-*

gprobe-linux-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/gprobe
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-linux-* | grep 386

gprobe-linux-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/gprobe
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-linux-* | grep amd64

gprobe-linux-arm: gprobe-linux-arm-5 gprobe-linux-arm-6 gprobe-linux-arm-7 gprobe-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-linux-* | grep arm

gprobe-linux-arm-5:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/gprobe
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-linux-* | grep arm-5

gprobe-linux-arm-6:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/gprobe
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-linux-* | grep arm-6

gprobe-linux-arm-7:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/gprobe
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-linux-* | grep arm-7

gprobe-linux-arm64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/gprobe
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-linux-* | grep arm64

gprobe-linux-mips:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/gprobe
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-linux-* | grep mips

gprobe-linux-mipsle:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/gprobe
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-linux-* | grep mipsle

gprobe-linux-mips64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/gprobe
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-linux-* | grep mips64

gprobe-linux-mips64le:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/gprobe
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-linux-* | grep mips64le

gprobe-darwin: gprobe-darwin-386 gprobe-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-darwin-*

gprobe-darwin-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/gprobe
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-darwin-* | grep 386

gprobe-darwin-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/gprobe
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-darwin-* | grep amd64

gprobe-windows: gprobe-windows-386 gprobe-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-windows-*

gprobe-windows-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/gprobe
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-windows-* | grep 386

gprobe-windows-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/gprobe
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gprobe-windows-* | grep amd64
