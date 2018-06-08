build:
	go run ./build/*.go build

deps:
	go run ./build/*.go deps

install:
	go run ./build/*.go install

test:
	go run ./build/*.go test
