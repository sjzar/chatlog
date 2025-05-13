BINARY_NAME := chatlog
GO := go
ifeq ($(VERSION),)
	VERSION := $(shell git describe --tags --always --dirty="-dev")
endif
LDFLAGS := -ldflags '-X "github.com/sjzar/chatlog/pkg/version.Version=$(VERSION)" -w -s'

PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64 \
	windows/arm64

UPX_PLATFORMS := \
	darwin/amd64 \
	windows/386 \
	windows/amd64

.PHONY: all clean lint tidy test build crossbuild upx

all: clean lint tidy test build

clean:
	@echo "🧹 Cleaning..."
	@rm -rf bin/

lint:
	@echo "🕵️‍♂️ Running linters..."
	golangci-lint run ./...

tidy:
	@echo "🧼 Tidying up dependencies..."
	$(GO) mod tidy

test:
	@echo "🧪 Running tests..."
	$(GO) test ./... -cover

build:
	@echo "🔨 Building for current platform..."
	CGO_ENABLED=1 $(GO) build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME) main.go

crossbuild: clean
	@echo "🌍 Building for multiple platforms..."
	for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		float=$$(echo $$platform | cut -d/ -f3); \
		output_name=bin/chatlog_$${os}_$${arch}; \
		[ "$$float" != "" ] && output_name=$$output_name_$$float; \
		# 👉 仅 Windows 添加 .exe 后缀
		[ "$$os" = "windows" ] && output_name=$${output_name}.exe; \
		echo "🔨 Building for $$os/$$arch..."; \
		echo "🔨 Building for $$output_name..."; \
		if [ "$$os" = "windows" ]; then \
			CC=x86_64-w64-mingw32-gcc; \
		else \
			CC=gcc; \
		fi; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=1 CC=$$CC GOARM=$$float $(GO) build -trimpath $(LDFLAGS) -o $$output_name main.go ; \
		if [ "$(ENABLE_UPX)" = "1" ] && echo "$(UPX_PLATFORMS)" | grep -q "$$os/$$arch"; then \
			echo "⚙️ Compressing binary $$output_name..." && upx --best $$output_name; \
		fi; \
	done