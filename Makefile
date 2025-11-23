.PHONY: build run test integration-test load-test clean docker-up docker-down lint format

build:
	go build -o bin/reviewer-service ./cmd/server

run:
	go run ./cmd/server/main.go

test:
	go test -v ./tests/

integration-test:
	go test -v ./tests/ -run Integration

load-test:
	go test -v ./tests/ -run TestConcurrent
	go test -v ./tests/ -run TestResponseTime
	go test -v ./tests/ -run TestBulkDeactivate
	go test -bench=. ./tests/ -benchmem

clean:
	rm -rf bin/

docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down -v

docker-logs:
	docker-compose logs -f

lint:
	golangci-lint run ./...

format:
	gofmt -w .
	goimports -w .