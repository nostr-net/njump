.PHONY: all build dev deploy clean templ tailwind protobuf prettier install-deps

# Add node_modules binaries to PATH
export PATH := ./node_modules/.bin:$(PATH)

# Source files
TEMPL_FILES := $(wildcard *.templ)
TEMPL_GO_FILES := $(TEMPL_FILES:.templ=_templ.go)
GO_FILES := $(wildcard *.go)
CSS_INPUT := base.css
CSS_OUTPUT := static/tailwind-bundle.min.css

# Build timestamp for cache busting
BUILD_TS := $(shell date '+%s')

# Default target
all: build

# Install dependencies
install-deps:
	go mod download
	npm install tailwindcss

# Generate templ files
templ: $(TEMPL_GO_FILES)

%_templ.go: %.templ
	templ generate -f $<

# Generate tailwind CSS
tailwind: $(CSS_OUTPUT)

$(CSS_OUTPUT): $(CSS_INPUT) tailwind.config.js $(TEMPL_FILES)
	npx tailwind -i $(CSS_INPUT) -o $(CSS_OUTPUT) --minify

# Build the binary
build: templ tailwind
	go build -o ./njump

# Development mode with file watching
dev: install-deps
	fd 'go|templ|base.css' | entr -r bash -c 'templ generate && go build -o /tmp/njump && TAILWIND_DEBUG=true PORT=3001 /tmp/njump'

# Deploy to target server (usage: make deploy TARGET=user@host)
deploy: templ tailwind
ifndef TARGET
	$(error TARGET is not set. Usage: make deploy TARGET=user@host)
endif
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=$$(which musl-gcc) \
		go build -tags='libsecp256k1' \
		-ldflags="-linkmode external -extldflags '-static' -X main.compileTimeTs=$(BUILD_TS)" \
		-o ./njump
	scp njump $(TARGET):njump/njump-new
	ssh $(TARGET) 'systemctl stop njump'
	ssh $(TARGET) 'mv njump/njump-new njump/njump'
	ssh $(TARGET) 'systemctl start njump'

# Generate protobuf
protobuf:
	protoc --proto_path=. --go_out=. --go_opt=paths=source_relative internal.proto

# Format templates
prettier:
	prettier -w templates/*.html

# Clean generated files
clean:
	rm -f $(TEMPL_GO_FILES)
	rm -f $(CSS_OUTPUT)
	rm -f ./njump
