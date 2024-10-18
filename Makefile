BINARY_NAME=gosvm

.PHONY: build
build:
	go build -o build/${BINARY_NAME} ./...

.PHONY: clean
clean:
	go clean
	rm build/${BINARY_NAME}

.PHONY: dep
dep:
	go mod download
	go mod verify
