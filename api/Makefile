.PHONY: run build test seed clean

run:
	go run main.go

build:
	go build -o bin/api main.go

test:
	go test ./...

seed:
	go run scripts/seed.go

clean:
	rm -rf bin/
	go clean

tidy:
	go mod tidy

