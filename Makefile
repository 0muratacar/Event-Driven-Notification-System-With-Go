.PHONY: build run test lint migrate-up migrate-down docker-up docker-down clean

APP_NAME := notifier
BUILD_DIR := bin

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/notifier

run: build
	./$(BUILD_DIR)/$(APP_NAME)

test:
	go test -v -race -count=1 ./...

test-short:
	go test -v -short -race ./...

lint:
	golangci-lint run ./...

migrate-up:
	migrate -path migrations -database "$(POSTGRES_DSN)" up

migrate-down:
	migrate -path migrations -database "$(POSTGRES_DSN)" down 1

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f notifier

clean:
	rm -rf $(BUILD_DIR)

tidy:
	go mod tidy
