.PHONY: build run clean

build:
	go build -o bin/ai-gateway ./cmd/ai-gateway

run: build
	./bin/ai-gateway

clean:
	rm -rf bin/