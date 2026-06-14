.PHONY: build run test tidy deps

build:
	go build -o ./bin/radius-manager ./main.go

run:
	go run ./main.go

deps:
	go mod download

tidy:
	go mod tidy

test:
	go test ./... -v

docker-build:
	docker build -t radius-manager-backend .
